package controller

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func NewCommand() *cobra.Command {
	var leaderElect bool

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Oblik operator",
		Run: func(cmd *cobra.Command, args []string) {
			go func() {
				handleSignals()
			}()
			Run(leaderElect)
		},
	}

	cmd.Flags().BoolVar(&leaderElect, "leader-elect", true, "Enable leader election for controller manager.")
	return cmd
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	klog.Info("Received termination signal, shutting down gracefully...")

	// Perform cleanup

	os.Exit(0)
}
