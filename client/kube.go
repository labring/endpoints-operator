// Copyright © 2022 The sealyun Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"github.com/sealyun/endpoints-operator/library/file"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path"
)

type Client interface {
	Kubernetes() kubernetes.Interface
	KubernetesDynamic() dynamic.Interface
	Config() *rest.Config
}

type kubernetesClient struct {
	// kubernetes client interface
	k8s        kubernetes.Interface
	k8sDynamic dynamic.Interface
	// discovery client
	config *rest.Config
}

type KubernetesOptions struct {
	// kubernetes clientset qps
	// +optional
	QPS float32 `json:"qps,omitempty" yaml:"qps"`
	// kubernetes clientset burst
	// +optional
	Burst      int `json:"burst,omitempty" yaml:"burst"`
	Kubeconfig string
	Master     string
	Config     *rest.Config `json:"-" yaml:"-"`
}

// NewKubernetesOptions returns a `zero` instance
func NewKubernetesOptions(kubeConfig, master string) *KubernetesOptions {
	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if kubeConfig == "" && kubeconfigPath != "" {
		kubeConfig = kubeconfigPath
	}
	if kubeconfigPath == "" {
		kubeConfig = path.Join(file.GetUserHomeDir(), clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName)
	}
	return &KubernetesOptions{
		QPS:        1e6,
		Burst:      1e6,
		Kubeconfig: kubeConfig,
		Master:     master,
	}
}

// NewKubernetesClient creates a KubernetesClient
func NewKubernetesClient(options *KubernetesOptions) Client {
	config := options.Config
	var err error
	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags(options.Master, options.Kubeconfig)
		if err != nil {
			return nil
		}
	}
	config.QPS = options.QPS
	config.Burst = options.Burst
	var k kubernetesClient
	k.k8s = kubernetes.NewForConfigOrDie(config)
	k.k8sDynamic = dynamic.NewForConfigOrDie(config)
	k.config = config

	return &k
}

func (k *kubernetesClient) Kubernetes() kubernetes.Interface {
	return k.k8s
}

func (k *kubernetesClient) Config() *rest.Config {
	return k.config
}

func (k *kubernetesClient) KubernetesDynamic() dynamic.Interface {
	return k.k8sDynamic
}
