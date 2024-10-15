package main

import (
	"fmt"
	"os"

	"github.com/SocialGouv/oblik/pkg/cli"
	"github.com/SocialGouv/oblik/pkg/controller"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)

	var rootCmd = cli.NewCommand()

	rootCmd.AddCommand(controller.NewCommand())

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
