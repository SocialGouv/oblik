package controller

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "oblik",
		Short: "Oblik operator",
		Run:   Cmd,
	}
}

func Cmd(cmd *cobra.Command, args []string) {
	go func() {
		handleSignals()
	}()
	Run()
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Received termination signal, shutting down gracefully...")

	// Perform cleanup

	os.Exit(0)
}
