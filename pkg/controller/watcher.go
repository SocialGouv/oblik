package controller

import (
	"context"
	"math/rand"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
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

func updateDeployment(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	deploymentName := targetRef.Name

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error fetching deployment: %s", err.Error())
		return
	}

	updateContainerResources(deployment.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(deployment, "apps/v1", "Deployment")
	if err != nil {
		klog.Errorf("Error creating patch: %s", err.Error())
		return
	}

	force := true
	_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), deploymentName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "vpa-operator",
		Force:        &force, // Force the apply to take ownership of the fields
	})
	if err != nil {
		klog.Errorf("Error applying patch to deployment: %s", err.Error())
	}
}

func updateStatefulSet(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef
	statefulSetName := targetRef.Name

	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error fetching stateful set: %s", err.Error())
		return
	}

	updateContainerResources(statefulSet.Spec.Template.Spec.Containers, vpa, vcfg)

	patchData, err := createPatch(statefulSet, "apps/v1", "StatefulSet")
	if err != nil {
		klog.Errorf("Error creating patch: %s", err.Error())
		return
	}

	_, err = clientset.AppsV1().StatefulSets(namespace).Patch(context.TODO(), statefulSetName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "oblik-operator",
	})
	if err != nil {
		klog.Errorf("Error applying patch to stateful set: %s", err.Error())
	}
}

func updateContainerResources(containers []corev1.Container, vpa *vpa.VerticalPodAutoscaler, vcfg *VPAOblikConfig) {
	for _, container := range containers {

		if vcfg.RequestCPUApplyMode == ApplyModeEnforce {
			var newCPURequests resource.Quantity
			for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
				if containerRecommendation.ContainerName == container.Name {
					newCPURequests = *containerRecommendation.Target.Cpu()
					break
				}
			}
			container.Resources.Requests[corev1.ResourceCPU] = newCPURequests
		}
		if vcfg.LimitCPUApplyMode == ApplyModeEnforce {
			newCPULimits := calculateNewLimitValue(container.Resources.Requests[corev1.ResourceCPU], vcfg.LimitCPUCalculatorAlgo, vcfg.LimitCPUCalculatorValue)
			container.Resources.Limits[corev1.ResourceCPU] = newCPULimits
		}

		if vcfg.RequestMemoryApplyMode == ApplyModeEnforce {
			var newMemoryRequests resource.Quantity
			for _, containerRecommendation := range vpa.Status.Recommendation.ContainerRecommendations {
				if containerRecommendation.ContainerName == container.Name {
					newMemoryRequests = *containerRecommendation.Target.Memory()
					break
				}
			}
			container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequests
		}
		if vcfg.LimitMemoryApplyMode == ApplyModeEnforce {
			newMemoryLimits := calculateNewLimitValue(container.Resources.Requests[corev1.ResourceMemory], vcfg.LimitMemoryCalculatorAlgo, vcfg.LimitMemoryCalculatorValue)
			container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimits
		}

	}
}
