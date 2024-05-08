package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/SocialGouv/oblik/pkg/controller"
)

func main() {
	handleSignals()
	controller.Run()
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Received termination signal, shutting down gracefully...")

	// Perform cleanup

	os.Exit(0)
}
