package controller

import (
	"context"
	"math/rand"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func watchVPAs(ctx context.Context, clientset *kubernetes.Clientset, vpaClientset *vpaclientset.Clientset) {
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
				handleVPA(clientset, vpaClientset, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleVPA(clientset, vpaClientset, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				klog.Info("VPA deleted")
			},
		},
	)

	klog.Info("Starting VPA watcher...")
	controller.Run(ctx.Done())
}

func handleVPA(clientset *kubernetes.Clientset, vpaClientset *vpaclientset.Clientset, obj interface{}) {
	vpa, ok := obj.(*vpa.VerticalPodAutoscaler)
	if !ok {
		klog.Error("Could not cast to VPA object")
		return
	}

	klog.Infof("Handling VPA: %s/%s", vpa.Namespace, vpa.Name)

	applyRecommendations(clientset, vpa)
}

func applyRecommendations(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler) {
	cronMutex.Lock()
	defer cronMutex.Unlock()

	vcfg := createVPAOblikConfig(vpa)

	key := vcfg.Key

	klog.Infof("Scheduling VPA recommendations for %s with cron: %s", key, vcfg.CronExpr)

	if entryID, exists := cronJobs[key]; exists {
		cronScheduler.Remove(entryID)
	}

	entryID, err := cronScheduler.AddFunc(vcfg.CronExpr, func() {
		randomDelay := time.Duration(rand.Int63n(vcfg.CronMaxRandomDelay.Nanoseconds()))
		time.Sleep(randomDelay)
		klog.Infof("Applying VPA recommendations for %s with cron: %s", key, vcfg.CronExpr)
		applyVPARecommendations(clientset, vpa, vcfg)
	})
	if err != nil {
		klog.Errorf("Error scheduling cron job: %s", err.Error())
		return
	}
	cronJobs[key] = entryID
}

func applyVPARecommendations(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	targetRef := vpa.Spec.TargetRef
	switch targetRef.Kind {
	case "Deployment":
		updateDeployment(clientset, vpa, vcfg)
	case "StatefulSet":
		updateStatefulSet(clientset, vpa, vcfg)
	}
}
