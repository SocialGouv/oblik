package watcher

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/SocialGouv/oblik/pkg/config"
	"github.com/SocialGouv/oblik/pkg/reporting"
	"github.com/SocialGouv/oblik/pkg/target"
	cron "github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var (
	CronScheduler = cron.New()
	cronJobs      = make(map[string]cron.EntryID)
	cronMutex     sync.Mutex
)

func WatchVPAs(ctx context.Context, clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpaClientset *vpaclientset.Clientset) {
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
				handleVPA(clientset, dynamicClient, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleVPA(clientset, dynamicClient, newObj)
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

func handleVPA(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, obj interface{}) {
	vpa, ok := obj.(*vpa.VerticalPodAutoscaler)
	if !ok {
		klog.Error("Could not cast to VPA object")
		return
	}

	klog.Infof("Handling VPA: %s/%s", vpa.Namespace, vpa.Name)

	scheduleVPA(clientset, dynamicClient, vpa)
}

func scheduleVPA(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpa *vpa.VerticalPodAutoscaler) {
	cronMutex.Lock()
	defer cronMutex.Unlock()

	vcfg := config.CreateVpaWorkloadCfg(vpa)

	key := vcfg.Key

	klog.Infof("Scheduling VPA recommendations for %s with cron: %s", key, vcfg.CronExpr)

	if entryID, exists := cronJobs[key]; exists {
		CronScheduler.Remove(entryID)
	}

	entryID, err := CronScheduler.AddFunc(vcfg.CronExpr, func() {
		randomDelay := time.Duration(rand.Int63n(vcfg.CronMaxRandomDelay.Nanoseconds()))
		time.Sleep(randomDelay)
		klog.Infof("Applying VPA recommendations for %s with cron: %s", key, vcfg.CronExpr)
		applyVPARecommendations(clientset, dynamicClient, vpa, vcfg)
	})
	if err != nil {
		klog.Errorf("Error scheduling cron job: %s", err.Error())
		return
	}
	cronJobs[key] = entryID
}

func applyVPARecommendations(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpa *vpa.VerticalPodAutoscaler, vcfg *config.VpaWorkloadCfg) {
	targetRef := vpa.Spec.TargetRef
	var updates *[]reporting.Update
	var err error
	switch targetRef.Kind {
	case "Deployment":
		updates, err = target.UpdateDeployment(clientset, vpa, vcfg)
	case "StatefulSet":
		updates, err = target.UpdateStatefulSet(clientset, vpa, vcfg)
	case "CronJob":
		updates, err = target.UpdateCronJob(clientset, vpa, vcfg)
	case "Cluster":
		if targetRef.APIVersion == "postgresql.cnpg.io/v1" {
			updates, err = target.UpdateCluster(dynamicClient, vpa, vcfg)
		} else {
			klog.Warningf("Unsupported Cluster kind from apiVersion: %s", targetRef.APIVersion)
			return
		}
	}
	if err != nil {
		klog.Errorf("Failed to apply updates for %s: %s", vcfg.Key, err.Error())
		return
	}
	reporting.ReportUpdated(*updates, vcfg)
}
