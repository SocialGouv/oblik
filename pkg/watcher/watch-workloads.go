package watcher

import (
	"context"
	"strings"
	"time"

	"github.com/SocialGouv/oblik/pkg/client"
	ovpa "github.com/SocialGouv/oblik/pkg/vpa"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func WatchWorkloads(ctx context.Context, kubeClients *client.KubeClients) {
	clientset := kubeClients.Clientset
	dynamicClient := kubeClients.DynamicClient
	vpaClientset := kubeClients.VpaClientset
	config := kubeClients.RestConfig

	labelSelector := labels.SelectorFromSet(labels.Set{"oblik.socialgouv.io/enabled": "true"})

	deploymentWatcher := createWatcher(ctx, clientset, dynamicClient, vpaClientset,
		cache.NewFilteredListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", corev1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}),
		&appsv1.Deployment{})

	statefulSetWatcher := createWatcher(ctx, clientset, dynamicClient, vpaClientset,
		cache.NewFilteredListWatchFromClient(clientset.AppsV1().RESTClient(), "statefulsets", corev1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}),
		&appsv1.StatefulSet{})

	cronJobWatcher := createWatcher(ctx, clientset, dynamicClient, vpaClientset,
		cache.NewFilteredListWatchFromClient(clientset.BatchV1().RESTClient(), "cronjobs", corev1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}),
		&batchv1.CronJob{})

	daemonSetWatcher := createWatcher(ctx, clientset, dynamicClient, vpaClientset,
		cache.NewFilteredListWatchFromClient(clientset.AppsV1().RESTClient(), "daemonsets", corev1.NamespaceAll, func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector.String()
		}),
		&appsv1.DaemonSet{})

	klog.Info("Starting Workloads watchers...")
	go deploymentWatcher.Run(ctx.Done())
	go statefulSetWatcher.Run(ctx.Done())
	go cronJobWatcher.Run(ctx.Done())
	go daemonSetWatcher.Run(ctx.Done())

	// Check if CNPG CRD exists before creating the watcher
	cnpgCRDExists, err := checkCRDExists(ctx, config, "clusters.postgresql.cnpg.io")
	if err != nil {
		klog.Errorf("Error checking CNPG CRD: %v", err)
	} else if cnpgCRDExists {
		cnpgClusterWatcher := createWatcher(ctx, clientset, dynamicClient, vpaClientset,
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.LabelSelector = labelSelector.String()
					return dynamicClient.Resource(schema.GroupVersionResource{
						Group:    "postgresql.cnpg.io",
						Version:  "v1",
						Resource: "clusters",
					}).Namespace(corev1.NamespaceAll).List(ctx, options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.LabelSelector = labelSelector.String()
					return dynamicClient.Resource(schema.GroupVersionResource{
						Group:    "postgresql.cnpg.io",
						Version:  "v1",
						Resource: "clusters",
					}).Namespace(corev1.NamespaceAll).Watch(ctx, options)
				},
			},
			&unstructured.Unstructured{})
		go cnpgClusterWatcher.Run(ctx.Done())
		klog.Info("CNPG Cluster watcher started")
	} else {
		klog.Info("CNPG CRD not found, skipping CNPG Cluster watcher")
	}

	<-ctx.Done()
}

func checkCRDExists(ctx context.Context, config *rest.Config, crdName string) (bool, error) {
	apiextensionsClientset, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return false, err
	}

	_, err = apiextensionsClientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func createWatcher(ctx context.Context, clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, vpaClientset *vpaclientset.Clientset, lw cache.ListerWatcher, objType runtime.Object) cache.Controller {
	_, controller := cache.NewInformer(
		lw,
		objType,
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ovpa.AddVPA(clientset, dynamicClient, vpaClientset, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				ovpa.UpdateVPA(clientset, dynamicClient, vpaClientset, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				ovpa.DeleteVPA(vpaClientset, obj)
			},
		},
	)
	return controller
}
