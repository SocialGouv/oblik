package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
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

const defaultCron = "0 2 * * *"
const defaultCronAddRandomMax = "120m"

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

	klog.Infof("Handling VPA: %s", vpa.Name)

	applyRecommendations(clientset, vpa)
}

func applyRecommendations(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler) {
	annotations := vpa.Annotations
	if annotations == nil {
		klog.Info("No annotations found, skipping")
		return
	}

	cronExpr := annotations["oblik.socialgouv.io/cron"]
	if cronExpr == "" {
		cronExpr = getEnv("OBLIK_DEFAULT_CRON", defaultCron)
	}

	cronAddRandomMax := annotations["oblik.socialgouv.io/cron-add-random-max"]
	if cronAddRandomMax == "" {
		cronAddRandomMax = getEnv("OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX", defaultCronAddRandomMax)
	}
	maxRandomDelay := parseDuration(cronAddRandomMax, 120*time.Minute)

	cpuRecoApplyMode := annotations["oblik.socialgouv.io/cpu-reco-apply-mode"]
	memoryRecoApplyMode := annotations["oblik.socialgouv.io/memory-reco-apply-mode"]

	limitCPUApplyMode := annotations["oblik.socialgouv.io/limit-cpu-apply-mode"]
	limitMemoryApplyMode := annotations["oblik.socialgouv.io/limit-memory-apply-mode"]

	limitCPUCalculatorAlgo := annotations["oblik.socialgouv.io/limit-cpu-calculator-algo"]
	limitMemoryCalculatorAlgo := annotations["oblik.socialgouv.io/limit-memory-calculator-algo"]

	limitMemoryCalculatorValue := annotations["oblik.socialgouv.io/limit-memory-calculator-value"]
	limitCPUCalculatorValue := annotations["oblik.socialgouv.io/limit-cpu-calculator-value"]

	if cpuRecoApplyMode == "" {
		cpuRecoApplyMode = "enforce"
	}
	if memoryRecoApplyMode == "" {
		memoryRecoApplyMode = "enforce"
	}
	if limitCPUApplyMode == "" {
		limitCPUApplyMode = "enforce"
	}
	if limitMemoryApplyMode == "" {
		limitMemoryApplyMode = "enforce"
	}

	if limitCPUCalculatorAlgo == "" {
		limitCPUCalculatorAlgo = getEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_ALGO", "ratio")
	}
	if limitMemoryCalculatorAlgo == "" {
		limitMemoryCalculatorAlgo = getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_ALGO", "ratio")
	}
	if limitCPUCalculatorValue == "" {
		limitCPUCalculatorValue = getEnv("OBLIK_DEFAULT_LIMIT_CPU_CALCULATOR_VALUE", "1")
	}
	if limitMemoryCalculatorValue == "" {
		limitMemoryCalculatorValue = getEnv("OBLIK_DEFAULT_LIMIT_MEMORY_CALCULATOR_VALUE", "1")
	}

	klog.Infof("Applying VPA recommendations for %s with cron: %s, maxRandomDelay: %s, cpuRecoApplyMode: %s, memoryRecoApplyMode: %s, limitMemoryApplyMode: %s, limitCPUApplyMode: %s, limitCPUCalculatorAlgo: %s, limitMemoryCalculatorAlgo: %s, limitMemoryCalculatorValue: %s, limitCPUCalculatorValue: %s",
		vpa.Name, cronExpr, maxRandomDelay, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue)

	cronMutex.Lock()
	defer cronMutex.Unlock()

	key := fmt.Sprintf("%s/%s", vpa.Namespace, vpa.Name)

	// Remove existing cron job if it exists
	if entryID, exists := cronJobs[key]; exists {
		cronScheduler.Remove(entryID)
	}

	// Schedule the application of recommendations
	entryID, err := cronScheduler.AddFunc(cronExpr, func() {
		randomDelay := time.Duration(rand.Int63n(maxRandomDelay.Nanoseconds()))
		time.Sleep(randomDelay)
		applyVPARecommendations(clientset, vpa, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue)
	})
	if err != nil {
		klog.Errorf("Error scheduling cron job: %s", err.Error())
		return
	}
	cronJobs[key] = entryID
}

func applyVPARecommendations(clientset *kubernetes.Clientset, vpa *vpa.VerticalPodAutoscaler, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue string) {
	// Get the target deployment or statefulset
	namespace := vpa.Namespace
	targetRef := vpa.Spec.TargetRef

	switch targetRef.Kind {
	case "Deployment":
		updateDeployment(clientset, namespace, targetRef.Name, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue)
	case "StatefulSet":
		updateStatefulSet(clientset, namespace, targetRef.Name, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue)
	}
}

func updateDeployment(clientset *kubernetes.Clientset, namespace, deploymentName, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue string) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error fetching deployment: %s", err.Error())
		return
	}

	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]

		// Apply CPU recommendations
		if cpuRecoApplyMode != "off" {
			if cpuRecoApplyMode == "enforce" {
				newCPURequests := calculateNewResourceValue(container.Resources.Requests[corev1.ResourceCPU], limitCPUCalculatorAlgo, limitCPUCalculatorValue)
				container.Resources.Requests[corev1.ResourceCPU] = newCPURequests
			}
			if limitCPUApplyMode == "enforce" {
				newCPULimits := calculateNewResourceValue(container.Resources.Limits[corev1.ResourceCPU], limitCPUCalculatorAlgo, limitCPUCalculatorValue)
				container.Resources.Limits[corev1.ResourceCPU] = newCPULimits
			}
		}

		// Apply memory recommendations
		if memoryRecoApplyMode != "off" {
			if memoryRecoApplyMode == "enforce" {
				newMemoryRequests := calculateNewResourceValue(container.Resources.Requests[corev1.ResourceMemory], limitMemoryCalculatorAlgo, limitMemoryCalculatorValue)
				container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequests
			}
			if limitMemoryApplyMode == "enforce" {
				newMemoryLimits := calculateNewResourceValue(container.Resources.Limits[corev1.ResourceMemory], limitMemoryCalculatorAlgo, limitMemoryCalculatorValue)
				container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimits
			}
		}
	}

	patchData, err := createPatch(deployment)
	if err != nil {
		klog.Errorf("Error creating patch: %s", err.Error())
		return
	}

	_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), deploymentName, types.ApplyPatchType, patchData, metav1.PatchOptions{
		FieldManager: "vpa-operator",
	})
	if err != nil {
		klog.Errorf("Error applying patch to deployment: %s", err.Error())
	}
}

func updateStatefulSet(clientset *kubernetes.Clientset, namespace, statefulSetName, cpuRecoApplyMode, memoryRecoApplyMode, limitMemoryApplyMode, limitCPUApplyMode, limitCPUCalculatorAlgo, limitMemoryCalculatorAlgo, limitMemoryCalculatorValue, limitCPUCalculatorValue string) {
	statefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), statefulSetName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error fetching stateful set: %s", err.Error())
		return
	}

	for i := range statefulSet.Spec.Template.Spec.Containers {
		container := &statefulSet.Spec.Template.Spec.Containers[i]

		// Apply CPU recommendations
		if cpuRecoApplyMode != "off" {
			newCPURequests := calculateNewResourceValue(container.Resources.Requests[corev1.ResourceCPU], limitCPUCalculatorAlgo, limitCPUCalculatorValue)
			newCPULimits := calculateNewResourceValue(container.Resources.Limits[corev1.ResourceCPU], limitCPUCalculatorAlgo, limitCPUCalculatorValue)

			if cpuRecoApplyMode == "enforce" {
				container.Resources.Requests[corev1.ResourceCPU] = newCPURequests
			}
			if limitCPUApplyMode == "enforce" {
				container.Resources.Limits[corev1.ResourceCPU] = newCPULimits
			}
		}

		// Apply memory recommendations
		if memoryRecoApplyMode != "off" {
			newMemoryRequests := calculateNewResourceValue(container.Resources.Requests[corev1.ResourceMemory], limitMemoryCalculatorAlgo, limitMemoryCalculatorValue)
			newMemoryLimits := calculateNewResourceValue(container.Resources.Limits[corev1.ResourceMemory], limitMemoryCalculatorAlgo, limitMemoryCalculatorValue)

			if memoryRecoApplyMode == "enforce" {
				container.Resources.Requests[corev1.ResourceMemory] = newMemoryRequests
			}
			if limitMemoryApplyMode == "enforce" {
				container.Resources.Limits[corev1.ResourceMemory] = newMemoryLimits
			}
		}
	}

	patchData, err := createPatch(statefulSet)
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

func parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		klog.Warningf("Error parsing duration: %s, using default: %s", err.Error(), defaultDuration)
		return defaultDuration
	}
	return duration
}

func calculateNewResourceValue(currentValue resource.Quantity, algo, valueStr string) resource.Quantity {
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		klog.Warningf("Error parsing calculator value: %s", err.Error())
		return currentValue
	}

	currentQuantity, ok := currentValue.AsInt64()
	if !ok {
		klog.Warningf("Error converting quantity to int64 for resource")
		return currentValue
	}

	newValue := currentValue.DeepCopy()
	switch algo {
	case "ratio":
		newValue = *resource.NewQuantity(int64(float64(currentQuantity)*value), currentValue.Format)
	case "margin":
		newValue = *resource.NewQuantity(currentQuantity+int64(value), currentValue.Format)
	default:
		klog.Warningf("Unknown calculator algorithm: %s", algo)
	}

	return newValue
}

func createPatch(obj interface{}) ([]byte, error) {
	var patchedObj interface{}

	switch t := obj.(type) {
	case *appsv1.Deployment:
		patchedObj = t.DeepCopy()
	case *appsv1.StatefulSet:
		patchedObj = t.DeepCopy()
	default:
		return nil, fmt.Errorf("unsupported type: %T", t)
	}

	jsonData, err := json.Marshal(patchedObj)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}
