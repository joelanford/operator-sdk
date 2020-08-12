// Copyright 2020 The Operator-SDK Authors
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

package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestLoadConfig(t *testing.T) {

	// create temp kubeconfig files
	recommended, err := ioutil.TempFile("/tmp", "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(recommended.Name())

	recommendedData := []byte(recommendedKubeconfig)
	err = ioutil.WriteFile(recommended.Name(), recommendedData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	clientcmd.RecommendedHomeFile = recommended.Name()

	override, err := ioutil.TempFile("/tmp", "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(override.Name())

	overrideData := []byte(overrideKubeconfig)
	err = ioutil.WriteFile(override.Name(), overrideData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	cases := []struct {
		name              string
		kubeconfigPath    string
		kubeconfigEnv     string
		namespace         string
		expectedHost      string
		expectedNamespace string
		expectedErr       error
	}{
		{name: "recommended file",
			expectedHost:      "https://recommended:6443",
			expectedNamespace: "recommended-goo"},
		{name: "recommended file override namespace",
			namespace:         "userspecified",
			expectedHost:      "https://recommended:6443",
			expectedNamespace: "userspecified"},
		{name: "kubeconfig and namespace override",
			kubeconfigPath:    override.Name(),
			namespace:         "userspecified",
			expectedHost:      "https://override:6443",
			expectedNamespace: "userspecified"},
		{name: "non-existent kubeconfig override",
			kubeconfigPath: "/tmp/doesnotexist",
			expectedErr:    os.ErrNotExist},
		{name: "kubeconfig override",
			kubeconfigPath:    override.Name(),
			expectedHost:      "https://override:6443",
			expectedNamespace: "override-goo"},
		{name: "kubeconfig env",
			kubeconfigEnv:     override.Name(),
			expectedHost:      "https://override:6443",
			expectedNamespace: "override-goo"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := Configuration{
				KubeconfigPath: c.kubeconfigPath,
				Namespace:      c.namespace,
			}
			if err := os.Setenv("KUBECONFIG", c.kubeconfigEnv); err != nil {
				t.Fatal(err)
			}
			err := cfg.Load()
			if !errors.Is(err, c.expectedErr) {
				t.Errorf("Wanted error %s, got: %s", c.expectedErr, err)
			}
			if err == nil {
				if cfg.Namespace != c.expectedNamespace {
					t.Errorf("Wanted namespace %s, got: %s", c.expectedNamespace, cfg.Namespace)
				}
				if cfg.RESTConfig.Host != c.expectedHost {
					t.Errorf("Wanted host %s, got: %s", c.expectedHost, cfg.RESTConfig.Host)
				}
			}
		})
	}
}

const recommendedKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    server: https://recommended:6443
  name: recommended
contexts:
- context:
    cluster: kubernetes
    namespace: foo
    user: kubernetes-admin
  name: dev
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
- context:
    cluster: recommended
    namespace: recommended-goo
    user: kubernetes-admin
  name: recommended
current-context: recommended
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
`

const overrideKubeconfig = `
apiVersion: v1
clusters:
- cluster:
    server: https://override:6443
  name: override
contexts:
- context:
    cluster: kubernetes
    namespace: foo
    user: kubernetes-admin
  name: dev
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
- context:
    cluster: override
    namespace: override-goo
    user: kubernetes-admin
  name: override
current-context: override
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
`
