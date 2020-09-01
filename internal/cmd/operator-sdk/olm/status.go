// Copyright 2019 The Operator-SDK Authors
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

package olm

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/operator-framework/operator-sdk/internal/client"
	"github.com/operator-framework/operator-sdk/internal/olm/installer"
)

func newStatusCmd() *cobra.Command {
	var (
		mgr     installer.Manager
		timeout time.Duration
	)
	cl := client.Client{
		NamespaceFlagInfo: &clientcmd.FlagInfo{
			LongName:    "olm-namespace",
			Default:     installer.DefaultOLMNamespace,
			Description: "namespace where OLM is installed",
		},
		SkipKubeconfigFlag: true,
	}
	cmd := &cobra.Command{
		Use:               "status",
		Short:             "Get the status of the Operator Lifecycle Manager installation in your cluster",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return cl.Load() },
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			mgr.Client = installer.NewClient(cl)
			if err := mgr.Status(ctx); err != nil {
				log.Fatalf("Failed to get OLM status: %s", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mgr.Version, "version", "", "version of OLM installed on cluster; if unset"+
		"operator-sdk attempts to auto-discover the version")
	cmd.Flags().DurationVar(&timeout, "timeout", installer.DefaultTimeout, "time to wait for the command to complete before failing")
	cl.BindFlags(cmd.Flags())
	return cmd
}
