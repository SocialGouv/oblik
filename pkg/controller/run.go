package controller

import (
	"os"

	"github.com/SocialGouv/oblik/pkg/client"
	"k8s.io/klog/v2"

	ctrl "sigs.k8s.io/controller-runtime"
)

func Run(leaderElect bool) {
	kubeClients := client.NewKubeClients()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection:   leaderElect,
		LeaderElectionID: "oblik-operator-leader-election",
	})
	if err != nil {
		klog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := mgr.Add(&webhookServerRunnable{
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
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
