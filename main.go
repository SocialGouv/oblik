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

	var rootCmd = controller.NewCommand()

	rootCmd.AddCommand(cli.NewCommand())

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
