package controller

import (
	"context"
	"os"

	oblikv1 "github.com/SocialGouv/oblik/pkg/apis/oblik/v1"
	"github.com/SocialGouv/oblik/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func Run(leaderElect bool, ctx context.Context) {
	// Set up controller-runtime logger
	log.SetLogger(zap.New())

	// Create the manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection:   leaderElect,
		LeaderElectionID: "oblik-operator-leader-election",
	})
	if err != nil {
		klog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Register ResourcesConfig types with the controller-runtime scheme
	schemeBuilder := runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(
			schema.GroupVersion{Group: oblikv1.GroupName, Version: oblikv1.Version},
			&oblikv1.ResourcesConfig{},
			&oblikv1.ResourcesConfigList{},
		)
		metav1.AddToGroupVersion(scheme, schema.GroupVersion{Group: oblikv1.GroupName, Version: oblikv1.Version})
		return nil
	})
	if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err, "unable to add ResourcesConfig to scheme")
		os.Exit(1)
	}

	// Register ResourcesConfig types with the client-go scheme
	client.AddToScheme()

	kubeClients := client.NewKubeClients()

	if err := mgr.Add(&serverRunnable{
		KubeClients: kubeClients,
	}); err != nil {
		klog.Error(err, "unable to add webhook server runnable")
		os.Exit(1)
	}

	if err := mgr.Add(&watcherRunnable{
		KubeClients: kubeClients,
	}); err != nil {
		klog.Error(err, "unable to add watcher runnable")
		os.Exit(1)
	}

	klog.Info("Starting Oblik Operator...")
	if err := mgr.Start(ctx); err != nil {
		klog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
