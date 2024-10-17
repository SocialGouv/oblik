package controller

import (
	"context"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var leaderElect bool

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Oblik operator",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())
			go handleSignals(ctx, cancel)
			Run(leaderElect, ctx)
		},
	}

	cmd.Flags().BoolVar(&leaderElect, "leader-elect", true, "Enable leader election for controller manager.")
	return cmd
}
