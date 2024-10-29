package cleaner

import (
	"context"
	"fmt"
	"strings"

	"github.com/SocialGouv/oblik/pkg/client"
	"github.com/SocialGouv/oblik/pkg/config"
	ovpa "github.com/SocialGouv/oblik/pkg/vpa"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/discovery"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
)

func CleanUpVPAs(ctx context.Context, kubeClients *client.KubeClients) {
	// Use VpaClientset to list all VPAs across all namespaces
	vpaList, err := kubeClients.VpaClientset.AutoscalingV1().VerticalPodAutoscalers("").List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Error failed to list VPAs: %s", err.Error())
	}

	for _, vpa := range vpaList.Items {
		if strings.HasPrefix(vpa.Name, config.VpaPrefix) {
			if err := processVPA(ctx, kubeClients, &vpa); err != nil {
				klog.Errorf("Error processing VPA %s: %s\n", vpa.Name, err.Error())
			}
		}
	}
}

func processVPA(ctx context.Context, kubeClients *client.KubeClients, vpa *vpav1.VerticalPodAutoscaler) error {
	targetRef := vpa.Spec.TargetRef
	if targetRef == nil {
		// Delete the VPA if TargetRef is nil
		ovpa.DeleteVPA(kubeClients.VpaClientset, vpa)
		return nil
	}

	// Create a discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(kubeClients.RestConfig)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Create a RESTMapper to map GVK to GVR
	cachedDiscoveryClient := cached.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)

	// Parse the APIVersion into Group and Version
	groupVersion, err := schema.ParseGroupVersion(targetRef.APIVersion)
	if err != nil {
		return fmt.Errorf("failed to parse APIVersion: %w", err)
	}

	gvk := schema.GroupVersionKind{
		Group:   groupVersion.Group,
		Version: groupVersion.Version,
		Kind:    targetRef.Kind,
	}

	// Map GVK to GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to map GVK to GVR: %w", err)
	}

	dynamicClient := kubeClients.DynamicClient

	// Get the dynamic resource interface for the target resource
	var resourceInterface dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		resourceInterface = dynamicClient.Resource(mapping.Resource).Namespace(vpa.Namespace)
	} else {
		resourceInterface = dynamicClient.Resource(mapping.Resource)
	}

	// Attempt to get the target resource
	_, err = resourceInterface.Get(ctx, targetRef.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Target does not exist; delete the VPA
			ovpa.DeleteVPA(kubeClients.VpaClientset, vpa)
			return nil
		} else {
			return fmt.Errorf("error fetching target: %w", err)
		}
	}
	return nil
}
