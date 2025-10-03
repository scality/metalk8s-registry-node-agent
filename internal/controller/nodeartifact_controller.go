/*
Copyright 2025 Scality.

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

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
)

// NodeArtifactReconciler reconciles a NodeArtifact object
type NodeArtifactReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	NodeName string
}

// +kubebuilder:rbac:groups=metalk8s.scality.com,resources=nodeartifacts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metalk8s.scality.com,resources=nodeartifacts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=metalk8s.scality.com,resources=nodeartifacts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeArtifact object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *NodeArtifactReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func getNodeName(o client.Object) string {
	if obj, ok := o.(*metalk8sv1alpha1.NodeArtifact); ok {
		return obj.Spec.NodeName
	}
	return ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeArtifactReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Predicate will ensure that NodeArtifact from event is related to the current Node
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return getNodeName(e.Object) == r.NodeName
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return getNodeName(e.ObjectOld) == r.NodeName
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return getNodeName(e.Object) == r.NodeName
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return getNodeName(e.Object) == r.NodeName
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&metalk8sv1alpha1.NodeArtifact{}, builder.WithPredicates(p)).
		Named("nodeartifact").
		Complete(r)
}
