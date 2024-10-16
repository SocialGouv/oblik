package watcher

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/target"
	cron "github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var (
	CronScheduler = cron.New()
	cronJobs      = make(map[string]cron.EntryID)
	cronMutex     sync.Mutex
)

func WatchVPAs(ctx context.Context, kubeClients *client.KubeClients) {
	vpaClientset := kubeClients.VpaClientset

	labelSelector := labels.SelectorFromSet(labels.Set{"oblik.socialgouv.io/enabled": "true"})
	watchlist := cache.NewFilteredListWatchFromClient(
		vpaClientset.AutoscalingV1().RESTClient(),
		"verticalpodautoscalers",
		metav1.NamespaceAll,
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		},
	)

	_, controller := cache.NewInformer(
		watchlist,
		&vpa.VerticalPodAutoscaler{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleVPA(kubeClients, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleVPA(kubeClients, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				klog.Info("VPA deleted")
				vpa, ok := obj.(*vpa.VerticalPodAutoscaler)
				if !ok {
					klog.Error("Could not cast to VPA object")
					return
				}
				key := fmt.Sprintf("%s/%s", vpa.Namespace, vpa.Name)
				if entryID, exists := cronJobs[key]; exists {
					CronScheduler.Remove(entryID)
				}
			},
		},
	)

	klog.Info("Starting VPA watcher...")
	controller.Run(ctx.Done())
}

func handleVPA(kubeClients *client.KubeClients, obj interface{}) {
	vpa, ok := obj.(*vpa.VerticalPodAutoscaler)
	if !ok {
		klog.Error("Could not cast to VPA object")
		return
	}

	klog.Infof("Handling VPA: %s/%s", vpa.Namespace, vpa.Name)

	scheduleVPA(kubeClients, vpa)
}

func scheduleVPA(kubeClients *client.KubeClients, vpaResource *vpa.VerticalPodAutoscaler) {
	cronMutex.Lock()
	defer cronMutex.Unlock()

	configurable := config.CreateConfigurable(vpaResource)
	scfg := config.CreateStrategyConfig(configurable)

	key := scfg.Key

	klog.Infof("Scheduling VPA recommendations for %s with cron: %s", key, scfg.CronExpr)

	if entryID, exists := cronJobs[key]; exists {
		CronScheduler.Remove(entryID)
	}

	entryID, err := CronScheduler.AddFunc(scfg.CronExpr, func() {
		nsecondsDelay := scfg.CronMaxRandomDelay.Nanoseconds()
		if nsecondsDelay != 0 {
			randomDelay := time.Duration(rand.Int63n(nsecondsDelay))
			time.Sleep(randomDelay)
		}
		klog.Infof("Applying VPA recommendations for %s with cron: %s", key, scfg.CronExpr)
		err := target.ApplyVPARecommendations(kubeClients, vpaResource, scfg)
		if err != nil {
			klog.Errorf("Error applying VPA recommendations: %s", err.Error())
		}
	})
	if err != nil {
		klog.Errorf("Error scheduling cron job: %s", err.Error())
		return
	}
	cronJobs[key] = entryID
}
