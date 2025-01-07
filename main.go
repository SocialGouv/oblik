package main

import (
	"fmt"
	"os"

	"github.com/SocialGouv/oblik/pkg/cli"
	"github.com/SocialGouv/oblik/pkg/controller"
	"github.com/spf13/viper"
)

func main() {
	var rootCmd = cli.NewCommand()

	rootCmd.AddCommand(controller.NewCommand())

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
