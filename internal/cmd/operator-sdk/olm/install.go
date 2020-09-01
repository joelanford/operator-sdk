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

	"github.com/operator-framework/operator-sdk/internal/client"
	"github.com/operator-framework/operator-sdk/internal/olm/installer"
)

func newInstallCmd() *cobra.Command {
	var (
		mgr     installer.Manager
		timeout time.Duration
	)
	cl := client.Client{
		SkipNamespaceFlag:  true,
		SkipKubeconfigFlag: true,
	}
	cmd := &cobra.Command{
		Use:               "install",
		Short:             "Install Operator Lifecycle Manager in your cluster",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return cl.Load() },
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			mgr.Client = installer.NewClient(cl)
			if err := mgr.Install(ctx); err != nil {
				log.Fatalf("Failed to install OLM version %q: %s", mgr.Version, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&mgr.Version, "version", installer.DefaultVersion, "version of OLM resources to install")
	cmd.Flags().DurationVar(&timeout, "timeout", installer.DefaultTimeout, "time to wait for the command to complete before failing")
	cl.BindFlags(cmd.Flags())
	return cmd
}
