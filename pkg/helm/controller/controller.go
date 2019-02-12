// Copyright 2018 The Operator-SDK Authors
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

package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	rpb "k8s.io/helm/pkg/proto/hapi/release"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	crthandler "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	crtpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/finalizer"
	"github.com/operator-framework/operator-sdk/pkg/helm/internal/types"
	"github.com/operator-framework/operator-sdk/pkg/helm/release"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
)

var log = logf.Log.WithName("helm.controller")

// WatchOptions contains the necessary values to create a new controller that
// manages helm releases in a particular namespace based on a GVK watch.
type WatchOptions struct {
	Namespace               string
	GVK                     schema.GroupVersionKind
	ManagerFactory          release.ManagerFactory
	ReconcilePeriod         time.Duration
	WatchDependentResources bool
}

func NewUninstallFinalizer(r *HelmOperatorReconciler) finalizer.Finalizer {
	return finalizer.FinalizeFunc(func(cr finalizer.Finalizable) error {
		o, ok := cr.(*unstructured.Unstructured)
		if !ok {
			return errors.New("unsupported finalizable type, expected *unstructured.Unstructured")
		}
		log := log.WithValues(
			"namespace", o.GetNamespace(),
			"name", o.GetName(),
			"apiVersion", o.GetAPIVersion(),
			"kind", o.GetKind(),
		)
		manager := r.ManagerFactory.NewManager(o)
		status := types.StatusFor(o)
		defer func() {
			o.Object["status"] = status
		}()

		uninstalledRelease, err := manager.UninstallRelease(context.TODO())
		if err != nil && err != release.ErrNotFound {
			log.Error(err, "Failed to uninstall release")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonUninstallError,
				Message: err.Error(),
			})
			_ = r.updateResourceStatus(o, status)
			return err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		if err == release.ErrNotFound {
			log.Info("Release not found, removing finalizer")
		} else {
			log.Info("Uninstalled release")
			if log.Enabled() {
				fmt.Println(diffutil.Diff(uninstalledRelease.GetManifest(), ""))
			}
			status.SetCondition(types.HelmAppCondition{
				Type:   types.ConditionDeployed,
				Status: types.StatusFalse,
				Reason: types.ReasonUninstallSuccessful,
			})
		}
		return r.updateResourceStatus(o, status)
	})
}

// Add creates a new helm operator controller and adds it to the manager
func Add(mgr manager.Manager, options WatchOptions) error {
	r := &HelmOperatorReconciler{
		Client:           mgr.GetClient(),
		FinalizerManager: finalizer.NewManager(),
		GVK:              options.GVK,
		ManagerFactory:   options.ManagerFactory,
		ReconcilePeriod:  options.ReconcilePeriod,
	}
	r.FinalizerManager.RegisterFinalizer("uninstall-helm-release", NewUninstallFinalizer(r))

	// Register the GVK with the schema
	mgr.GetScheme().AddKnownTypeWithName(options.GVK, &unstructured.Unstructured{})
	metav1.AddToGroupVersion(mgr.GetScheme(), options.GVK.GroupVersion())

	controllerName := fmt.Sprintf("%v-controller", strings.ToLower(options.GVK.Kind))
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(options.GVK)
	if err := c.Watch(&source.Kind{Type: o}, &crthandler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	if options.WatchDependentResources {
		watchDependentResources(mgr, r, c)
	}

	log.Info("Watching resource", "apiVersion", options.GVK.GroupVersion(), "kind", options.GVK.Kind, "namespace", options.Namespace, "reconcilePeriod", options.ReconcilePeriod.String())
	return nil
}

// watchDependentResources adds a release hook function to the HelmOperatorReconciler
// that adds watches for resources in released Helm charts.
func watchDependentResources(mgr manager.Manager, r *HelmOperatorReconciler, c controller.Controller) {
	owner := &unstructured.Unstructured{}
	owner.SetGroupVersionKind(r.GVK)

	dependentPredicate := crtpredicate.Funcs{
		// We don't need to reconcile dependent resource creation events
		// because dependent resources are only ever created during
		// reconciliation. Another reconcile would be redundant.
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*unstructured.Unstructured)
			log.V(1).Info("Skipping reconciliation for dependent resource creation", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
			return false
		},

		// Reconcile when a dependent resource is deleted so that it can be
		// recreated.
		DeleteFunc: func(e event.DeleteEvent) bool {
			o := e.Object.(*unstructured.Unstructured)
			log.V(1).Info("Reconciling due to dependent resource deletion", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
			return true
		},

		// Reconcile when a dependent resource is updated, so that it can
		// be patched back to the resource managed by the Helm release, if
		// necessary. Ignore updates that only change the status and
		// resourceVersion.
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*unstructured.Unstructured).DeepCopy()
			new := e.ObjectNew.(*unstructured.Unstructured).DeepCopy()

			delete(old.Object, "status")
			delete(new.Object, "status")
			old.SetResourceVersion("")
			new.SetResourceVersion("")

			if reflect.DeepEqual(old.Object, new.Object) {
				return false
			}
			log.V(1).Info("Reconciling due to dependent resource update", "name", new.GetName(), "namespace", new.GetNamespace(), "apiVersion", new.GroupVersionKind().GroupVersion(), "kind", new.GroupVersionKind().Kind)
			return true
		},
	}

	var m sync.RWMutex
	watches := map[schema.GroupVersionKind]struct{}{}
	releaseHook := func(release *rpb.Release) error {
		dec := yaml.NewDecoder(bytes.NewBufferString(release.GetManifest()))
		for {
			var u unstructured.Unstructured
			err := dec.Decode(&u.Object)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}

			gvk := u.GroupVersionKind()
			m.RLock()
			_, ok := watches[gvk]
			m.RUnlock()
			if ok {
				continue
			}

			restMapper := mgr.GetRESTMapper()
			depMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return err
			}
			ownerMapping, err := restMapper.RESTMapping(owner.GroupVersionKind().GroupKind(), owner.GroupVersionKind().Version)
			if err != nil {
				return err
			}

			depClusterScoped := depMapping.Scope.Name() == meta.RESTScopeNameRoot
			ownerClusterScoped := ownerMapping.Scope.Name() == meta.RESTScopeNameRoot

			if !ownerClusterScoped && depClusterScoped {
				m.Lock()
				watches[gvk] = struct{}{}
				m.Unlock()
				log.Info("Cannot watch cluster-scoped dependent resource for namespace-scoped owner. Changes to this dependent resource type will not be reconciled",
					"ownerApiVersion", r.GVK.GroupVersion(), "ownerKind", r.GVK.Kind, "apiVersion", gvk.GroupVersion(), "kind", gvk.Kind)
				continue
			}

			err = c.Watch(&source.Kind{Type: &u}, &crthandler.EnqueueRequestForOwner{OwnerType: owner}, dependentPredicate)
			if err != nil {
				return err
			}

			m.Lock()
			watches[gvk] = struct{}{}
			m.Unlock()
			log.Info("Watching dependent resource", "ownerApiVersion", r.GVK.GroupVersion(), "ownerKind", r.GVK.Kind, "apiVersion", gvk.GroupVersion(), "kind", gvk.Kind)
		}
	}
	r.releaseHook = releaseHook
}
