/*
SPDX-License-Identifier: Apache-2.0
Copyright Contributors to the Submariner project.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package subctl

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	//homedir "github.com/mitchellh/go-homedir"
	// "github.com/spf13/viper"
)

// var cfgFile string

var (
	kubeConfig   string
	kubeContext  string
	kubeContexts []string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "subctl",
	Short: "An installer for Submariner",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

/*
func init() {
	cobra.OnInitialize(initConfig)
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.subctl.yaml)")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// Search config in home directory with name ".subctl" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".subctl")
	}
	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
*/

func addKubeConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "absolute path(s) to the kubeconfig file(s)")
}

// addKubeContextFlag adds a "kubeconfig" flag and a single "kubecontext" flag that can be used once and only once
func addKubeContextFlag(cmd *cobra.Command) {
	addKubeConfigFlag(cmd)
	cmd.PersistentFlags().StringVar(&kubeContext, "kubecontext", "", "kubeconfig context to use")
}

// addKubeContextMultiFlag adds a "kubeconfig" flag and a "kubecontext" flag that can be specified multiple times (or comma separated)
func addKubeContextMultiFlag(cmd *cobra.Command, usage string) {
	addKubeConfigFlag(cmd)
	if usage == "" {
		usage = "comma-separated list of kubeconfig contexts to use, can be specified multiple times.\n" +
			"If none specified, all contexts referenced by the kubeconfig are used"
	}

	cmd.PersistentFlags().StringSliceVar(&kubeContexts, "kubecontexts", nil, usage)
}