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

package finalizer

import (
	"errors"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
)

var (
	ErrNotFound = errors.New("finalizer not found")
)

type Finalizer interface {
	Finalize(Finalizable) error
}

type FinalizeFunc func(Finalizable) error

func (f FinalizeFunc) Finalize(finalizable Finalizable) error {
	return f(finalizable)
}

type Finalizable interface {
	runtime.Object
	GetFinalizers() []string
	SetFinalizers([]string)
}

type Manager struct {
	mutex      sync.RWMutex
	finalizers map[string]Finalizer
}

func NewManager() Manager {
	return Manager{
		finalizers: make(map[string]Finalizer),
	}
}

func (fm *Manager) RegisterFinalizer(name string, f Finalizer) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	fm.finalizers[name] = f
}

func (fm *Manager) Finalize(finalizable Finalizable) error {
	toFinalize := finalizable.GetFinalizers()
	for i := 0; i < len(toFinalize); {
		name := toFinalize[i]
		fm.mutex.RLock()
		f, ok := fm.finalizers[name]
		fm.mutex.RUnlock()
		if ok {
			if err := f.Finalize(finalizable); err != nil {
				return err
			}
			toFinalize = append(toFinalize[0:i], toFinalize[i+1:len(toFinalize)]...)
			continue
		}
		i++
	}
	finalizable.SetFinalizers(toFinalize)
	return nil
}

func (fm *Manager) FinalizeOne(name string, finalizable Finalizable) error {
	fm.mutex.RLock()
	f, ok := fm.finalizers[name]
	fm.mutex.RUnlock()

	if !ok {
		return ErrNotFound
	}

	if err := f.Finalize(finalizable); err != nil {
		return err
	}

	toFinalize := finalizable.GetFinalizers()
	for i := 0; i < len(toFinalize); i++ {
		if name == toFinalize[i] {
			toFinalize = append(toFinalize[0:i], toFinalize[i+1:len(toFinalize)]...)
			break
		}
	}
	finalizable.SetFinalizers(toFinalize)
	return nil
}

func (fm *Manager) Reconcile(finalizable Finalizable) (bool, error) {
	finalizers := finalizable.GetFinalizers()
	existingFinalizers := make(map[string]struct{})
	for _, f := range finalizers {
		existingFinalizers[f] = struct{}{}
	}

	added := false
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()
	for f := range fm.finalizers {
		if _, ok := existingFinalizers[f]; !ok {
			added = true
			finalizers = append(finalizers, f)
		}
	}
	return added, nil
}

func (fm *Manager) ReconcileOne(name string, finalizable Finalizable) (bool, error) {
	finalizers := finalizable.GetFinalizers()
	for _, f := range finalizers {
		if f == name {
			return false, nil
		}
	}
	finalizers = append(finalizers, name)
	finalizable.SetFinalizers(finalizers)
	return true, nil
}
