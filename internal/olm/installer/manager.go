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

package installer

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultVersion = "latest"
	DefaultTimeout = time.Minute * 2
	// DefaultOLMNamespace is the namespace where OLM is installed
	DefaultOLMNamespace = "olm"
)

type Manager struct {
	Client  Client
	Version string
}

func (m *Manager) Install(ctx context.Context) error {
	status, err := m.Client.InstallVersion(ctx, m.Client.Namespace, m.Version)
	if err != nil {
		return err
	}

	log.Infof("Successfully installed OLM version %q", m.Version)
	fmt.Print("\n")
	fmt.Println(status)
	return nil
}

func (m *Manager) Uninstall(ctx context.Context) error {
	if version, err := m.Client.GetInstalledVersion(ctx, m.Client.Namespace); err != nil {
		if m.Version == "" {
			return fmt.Errorf("error getting installed OLM version (set --version to override the default version): %v", err)
		}
	} else if m.Version != "" {
		if version != m.Version {
			return fmt.Errorf("mismatched installed version %q vs. supplied version %q", version, m.Version)
		}
	} else {
		m.Version = version
	}

	if err := m.Client.UninstallVersion(ctx, m.Client.Namespace, m.Version); err != nil {
		return err
	}

	log.Infof("Successfully uninstalled OLM version %q", m.Version)
	return nil
}

func (m *Manager) Status(ctx context.Context) error {
	if version, err := m.Client.GetInstalledVersion(ctx, m.Client.Namespace); err != nil {
		if m.Version == "" {
			return fmt.Errorf("error getting installed OLM version (set --version to override the default version): %v", err)
		}
	} else if m.Version != "" {
		if version != m.Version {
			return fmt.Errorf("mismatched installed version %q vs. supplied version %q", version, m.Version)
		}
	} else {
		m.Version = version
	}

	status, err := m.Client.GetStatus(ctx, m.Client.Namespace, m.Version)
	if err != nil {
		return err
	}

	log.Infof("Successfully got OLM status for version %q", m.Version)
	fmt.Print("\n")
	fmt.Println(status)
	return nil
}
