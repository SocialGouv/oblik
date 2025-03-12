package watcher

import (
	"context"
	"time"

	oblikv1 "github.com/SocialGouv/oblik/pkg/apis/oblik/v1"
	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/resourcesconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

func WatchResourcesConfigs(ctx context.Context, kubeClients *client.KubeClients) {
	resourcesConfigClientset := kubeClients.ResourcesConfigClientset

	// Create a custom ListWatch for ResourcesConfigs
	watchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fields.Everything().String()
			return resourcesConfigClientset.OblikV1().List(ctx, metav1.NamespaceAll, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fields.Everything().String()
			return resourcesConfigClientset.OblikV1().Watch(ctx, metav1.NamespaceAll, options)
		},
	}

	_, controller := cache.NewInformer(
		watchlist,
		&oblikv1.ResourcesConfig{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				handleResourcesConfig(ctx, kubeClients, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				handleResourcesConfig(ctx, kubeClients, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				handleResourcesConfigDelete(ctx, kubeClients, obj)
			},
		},
	)

	klog.Info("Starting ResourcesConfigs watcher...")
	controller.Run(ctx.Done())
}

func handleResourcesConfig(ctx context.Context, kubeClients *client.KubeClients, obj interface{}) {
	rc, ok := obj.(*oblikv1.ResourcesConfig)
	if !ok {
		klog.Error("Could not cast to ResourcesConfig object")
		return
	}

	klog.Infof("Handling ResourcesConfig: %s/%s", rc.Namespace, rc.Name)

	err := resourcesconfig.SyncAnnotations(ctx, kubeClients, rc)
	if err != nil {
		if resourcesconfig.IsResourceNotFoundError(err) {
			// Log as warning instead of error when resource is not found
			klog.Warningf("Warning syncing annotations: %s", err.Error())
			// Update status with warning
			resourcesconfig.UpdateStatus(ctx, kubeClients, rc, false, err.Error())
			return
		}
		klog.Errorf("Error syncing annotations: %s", err.Error())
		// Update status with error
		resourcesconfig.UpdateStatus(ctx, kubeClients, rc, false, err.Error())
		return
	}

	// Update status with success
	resourcesconfig.UpdateStatus(ctx, kubeClients, rc, true, "")
}

func handleResourcesConfigDelete(ctx context.Context, kubeClients *client.KubeClients, obj interface{}) {
	rc, ok := obj.(*oblikv1.ResourcesConfig)
	if !ok {
		klog.Error("Could not cast to ResourcesConfig object")
		return
	}

	klog.Infof("Handling ResourcesConfig deletion: %s/%s", rc.Namespace, rc.Name)

	// If annotation mode is "replace", remove all oblik annotations from the target
	if rc.Spec.AnnotationMode != "merge" {
		err := resourcesconfig.RemoveAnnotations(ctx, kubeClients, rc)
		if err != nil {
			if resourcesconfig.IsResourceNotFoundError(err) {
				// Log as warning instead of error when resource is not found
				klog.Warningf("Warning removing annotations: %s", err.Error())
				return
			}
			klog.Errorf("Error removing annotations: %s", err.Error())
			return
		}
	}
}
