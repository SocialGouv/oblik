package controller

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
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
				handleVPA(clientset, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleVPA(clientset, newObj)
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
					cronScheduler.Remove(entryID)
				}
			},
		},
	)

	klog.Info("Starting VPA watcher...")
	controller.Run(ctx.Done())
}

func handleVPA(clientset *kubernetes.Clientset, obj interface{}) {
	vpa, ok := obj.(*vpa.VerticalPodAutoscaler)
	if !ok {
		klog.Error("Could not cast to VPA object")
		return
	}

	klog.Infof("Handling VPA: %s/%s", vpa.Namespace, vpa.Name)

	scheduleVPA(clientset, vpa)
}

func scheduleVPA(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler) {
	cronMutex.Lock()
	defer cronMutex.Unlock()

	vcfg := createVpaWorkloadCfg(vpa)

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

func getResourceValueText(updateType UpdateType, value resource.Quantity) string {
	switch updateType {
	case UpdateTypeMemoryLimit:
		return formatMemory(value)
	case UpdateTypeMemoryRequest:
		return formatMemory(value)
	default:
		return value.String()
	}
}

func reportUpdated(updates []Update, vcfg *VpaWorkloadCfg) {
	if len(updates) == 0 {
		return
	}
	klog.Infof("Updated: %s", vcfg.Key)
	for _, update := range updates {
		typeLabel := getUpdateTypeLabel(update.Type)
		oldValueText := getResourceValueText(update.Type, update.Old)
		newValueText := getResourceValueText(update.Type, update.New)
		klog.Infof("Setting %s to %s (previously %s) for %s container: %s", typeLabel, newValueText, oldValueText, vcfg.Key, update.ContainerName)
	}
	sendUpdatesToMattermost(updates, vcfg)
}

func applyVPARecommendations(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VpaWorkloadCfg) {
	targetRef := vpa.Spec.TargetRef
	var updates *[]Update
	var err error
	switch targetRef.Kind {
	case "Deployment":
		updates, err = updateDeployment(clientset, vpa, vcfg)
	case "StatefulSet":
		updates, err = updateStatefulSet(clientset, vpa, vcfg)
	case "CronJob":
		updates, err = updateCronJob(clientset, vpa, vcfg)
	}
	if err != nil {
		klog.Errorf("Failed to apply updates for %s: %s", vcfg.Key, err.Error())
		return
	}
	reportUpdated(*updates, vcfg)
}
