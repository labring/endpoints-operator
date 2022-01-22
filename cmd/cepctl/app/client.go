/*
Copyright 2022 The sealyun Authors.

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

package app

import (
	"fmt"
	"github.com/sealyun/endpoints-operator/library/file"
	"github.com/spf13/cobra"
	"os"
	"path"
)

var (
	// master is used to override the kubeconfig's URL to the apiserver.
	master string
	//kubeconfig is the path to a KubeConfig file.
	kubeconfig string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cepctl",
	Short: "cepctl is cli for cluster-endpoint",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", path.Join(file.GetUserHomeDir(), ".kube", "config"), "Path to kubeconfig file with authorization information (the master location can be overridden by the master flag).")
	rootCmd.PersistentFlags().StringVar(&master, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")

}
