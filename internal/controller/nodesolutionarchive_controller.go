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
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

const (
	FINALIZER_NAME = "metalk8s.scality.com/finalizer"
)

// NodeSolutionArchiveReconciler reconciles a NodeSolutionArchive object
type NodeSolutionArchiveReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	NodeName        string
	DownloadBaseURL string
	Container       containerInterface
	EventChan       chan event.GenericEvent
}

// +kubebuilder:rbac:groups=metalk8s.scality.com,resources=nodesolutionarchives,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metalk8s.scality.com,resources=nodesolutionarchives/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=metalk8s.scality.com,resources=nodesolutionarchives/finalizers,verbs=update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeSolutionArchive object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *NodeSolutionArchiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile NodeSolutionArchive")

	// 1. Load the NodeSolutionArchive by name
	nodeSolutionArchive := &metalk8sv1alpha1.NodeSolutionArchive{}
	if err := r.Get(ctx, req.NamespacedName, nodeSolutionArchive); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Add finalizer to deal with solution archive deletion
	//
	// examine DeletionTimestamp to determine if object is under deletion
	if nodeSolutionArchive.DeletionTimestamp.IsZero() {
		log.V(1).Info("instance is not being deleted")
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(nodeSolutionArchive, FINALIZER_NAME) {
			log.V(1).Info("Adding finalizer to NodeSolutionArchive")
			controllerutil.AddFinalizer(nodeSolutionArchive, FINALIZER_NAME)
			if err := r.Update(ctx, nodeSolutionArchive); err != nil {
				log.Error(err, "error adding finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		log.V(1).Info("NodeSolutionArchive is being deleted")
		if controllerutil.ContainsFinalizer(nodeSolutionArchive, FINALIZER_NAME) {
			// our finalizer is present, so lets handle any external dependency
			log.V(1).Info("Deleting external resources")
			if err := r.deleteSolutionArchiveResources(nodeSolutionArchive); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried.
				log.Error(err, "error deleting external Resources")
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			log.V(1).Info("Removing finalizer from NodeSolutionArchive")
			controllerutil.RemoveFinalizer(nodeSolutionArchive, FINALIZER_NAME)
			if err := r.Update(ctx, nodeSolutionArchive); err != nil {
				log.Error(err, "error removing finalizer")
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Retrieve the Status of the NodeSolutionArchive
	isAvailable := nodeSolutionArchive.Status.Available != nil && *nodeSolutionArchive.Status.Available

	// 3. Initialize the session
	solutionArchive := &domain.SolutionArchive{
		Name:    nodeSolutionArchive.Spec.Name,
		Version: nodeSolutionArchive.Spec.Version,
		Hash:    nodeSolutionArchive.Spec.Validation.Checksum.Value,
	}
	log.V(1).Info("Initializing session, if needed")
	_, err := r.Container.GetInitializeSessionUseCase().Execute(solutionArchive)
	if err != nil {
		log.Error(err, "error initializing session")
		return ctrl.Result{}, err
	}

	// 4. Check if an existing solution archive is already available on another node
	//    in order to download it
	if !isAvailable {
		log.V(1).Info("Checking if an existing solution archive is already available on another node")
		otherSolutionArchiveAvailable := false
		var urlToDownload string
		var crc32Checksum *uint32
		solutionArchiveNameVersion := library.GetSolutionArchiveNameVersion(
			nodeSolutionArchive.Spec.Name,
			nodeSolutionArchive.Spec.Version,
		)
		nodeSolutionArchiveList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
		if err := r.List(ctx, nodeSolutionArchiveList, client.MatchingFields{"SolutionArchiveNameVersion": solutionArchiveNameVersion}); err != nil {
			log.Error(err, "error listing node solution archives")
		}
		for _, item := range nodeSolutionArchiveList.Items {
			if item.Spec.NodeName != nodeSolutionArchive.Spec.NodeName && item.Status.Available != nil && *item.Status.Available {
				otherSolutionArchiveAvailable = true
				urlToDownload = item.Status.URL
				crc32Checksum = item.Status.CRC32Checksum
				break
			}
		}
		if otherSolutionArchiveAvailable {
			log.V(1).Info("Getting external solution archive from another node", "url", urlToDownload)
			err := r.Container.GetGetExternalSolutionArchiveUseCase().Execute(
				ctx,
				solutionArchive,
				urlToDownload,
				crc32Checksum,
			)
			if err != nil {
				// In some edge case, even if the solution archive is available on another node,
				// it may not be possible to get it.
				// Example: the watcher has not yet fully indexed the archive.
				// In this case, we will retry later.
				log.Error(err, "error getting external solution archive")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// 5. Checksum validation of the solution archive
	log.V(1).Info("Checking if the solution archive is valid")
	isValid, crc32Checksum, err := r.isValidSolutionArchive(nodeSolutionArchive)
	if err != nil {
		log.Error(err, "error validating solution archive")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	original := nodeSolutionArchive.DeepCopy()

	// Ensure we update the status in case of early return
	defer func() {
		if err := r.Status().Patch(ctx, nodeSolutionArchive, client.MergeFrom(original)); err != nil {
			log.Error(err, "unable to patch NodeSolutionArchive status")
		}
	}()

	if !isValid {
		log.V(1).Info("solution archive is not valid")
		nodeSolutionArchive.SetUnavailable()
		// Unmount the solution archive on file system
		err = r.Container.GetUnmountSolutionArchiveUseCase().Execute(solutionArchive)
		if err != nil {
			log.Error(err, "error unmounting solution archive")
			return ctrl.Result{}, err
		}
		nodeSolutionArchive.SetNotServed()
		nodeSolutionArchive.Status.URL = ""
		return ctrl.Result{}, nil
	}

	log.V(1).Info("solution archive is valid")
	downloadURL, err := url.JoinPath(r.DownloadBaseURL, nodeSolutionArchive.Spec.Name, nodeSolutionArchive.Spec.Version)
	if err != nil {
		log.Error(err, "error building the download URL")
		return ctrl.Result{}, err
	}
	nodeSolutionArchive.Status.URL = downloadURL
	nodeSolutionArchive.Status.CRC32Checksum = &crc32Checksum
	nodeSolutionArchive.SetAvailable()
	// Session may persist in case of manual upload of solution archive
	err = r.Container.GetRemoveSessionUseCase().Execute(solutionArchive)
	if err != nil {
		log.Error(err, "error removing session")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// 6. Mount the solution archive on file system
	log.V(1).Info("mounting the solution archive on file system")
	err = r.Container.GetMountSolutionArchiveUseCase().Execute(solutionArchive)
	if err != nil {
		log.Error(err, "error mounting solution archive")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	nodeSolutionArchive.SetServed()

	return ctrl.Result{}, nil
}

func (r *NodeSolutionArchiveReconciler) deleteSolutionArchiveResources(nodeSolutionArchive *metalk8sv1alpha1.NodeSolutionArchive) error {
	solutionArchive := &domain.SolutionArchive{
		Name:    nodeSolutionArchive.Spec.Name,
		Version: nodeSolutionArchive.Spec.Version,
		Hash:    nodeSolutionArchive.Spec.Validation.Checksum.Value,
	}

	err := r.Container.GetUnmountSolutionArchiveUseCase().Execute(solutionArchive)
	if err != nil {
		return err
	}

	err = r.Container.GetRemoveSolutionArchiveUseCase().Execute(solutionArchive)
	if err != nil {
		return err
	}

	return r.Container.GetRemoveSessionUseCase().Execute(solutionArchive)
}

func getNodeName(o client.Object) string {
	if obj, ok := o.(*metalk8sv1alpha1.NodeSolutionArchive); ok {
		return obj.Spec.NodeName
	}
	return ""
}

func (r *NodeSolutionArchiveReconciler) isValidSolutionArchive(nodeSolutionArchive *metalk8sv1alpha1.NodeSolutionArchive) (bool, uint32, error) {
	solutionArchive := &domain.SolutionArchive{
		Name:    nodeSolutionArchive.Spec.Name,
		Version: nodeSolutionArchive.Spec.Version,
		Hash:    nodeSolutionArchive.Spec.Validation.Checksum.Value,
	}
	return r.Container.GetValidateSolutionArchiveUseCase().Execute(solutionArchive)
}

// mapNodeSolutionArchiveToAvailableNodeSolutionArchive generate a []reconcile.Request based on a NodeSolutionArchive entry
// that is not associated to the current Node and that its status is "Available"
func mapNodeSolutionArchiveToAvailableNodeSolutionArchive(c client.Client) func(ctx context.Context, obj client.Object) []reconcile.Request {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		var result []reconcile.Request

		nodeSolutionArchive := obj.(*metalk8sv1alpha1.NodeSolutionArchive)
		if nodeSolutionArchive.Status.Available != nil && *nodeSolutionArchive.Status.Available {
			solutionArchiveNameVersion := library.GetSolutionArchiveNameVersion(nodeSolutionArchive.Spec.Name, nodeSolutionArchive.Spec.Version)
			nodeSolutionArchiveList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
			if err := c.List(ctx, nodeSolutionArchiveList, client.MatchingFields{"LocalSolutionArchiveNameVersion": solutionArchiveNameVersion}); err != nil {
				return result
			}
			if len(nodeSolutionArchiveList.Items) > 0 {
				item := nodeSolutionArchiveList.Items[0]
				itemAvailable := item.Status.Available != nil && *item.Status.Available
				if !itemAvailable {
					result = append(result, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name: item.Name,
						},
					})
				}
			}
		}
		return result
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeSolutionArchiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Predicate will ensure that NodeSolutionArchive from event is related to the current Node
	ensureSameNode := predicate.Funcs{
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

	ensureOtherNode := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return getNodeName(e.Object) != r.NodeName
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return getNodeName(e.ObjectOld) != r.NodeName
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return getNodeName(e.Object) != r.NodeName
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return getNodeName(e.Object) != r.NodeName
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&metalk8sv1alpha1.NodeSolutionArchive{}, builder.WithPredicates(ensureSameNode)).
		Watches(&metalk8sv1alpha1.NodeSolutionArchive{},
			handler.EnqueueRequestsFromMapFunc(mapNodeSolutionArchiveToAvailableNodeSolutionArchive(r.Client)),
			builder.WithPredicates(ensureOtherNode)).
		Named("nodesolutionarchive").
		WatchesRawSource(
			source.Channel(
				r.EventChan,
				handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
					na, ok := obj.(*metalk8sv1alpha1.NodeSolutionArchive)
					if !ok {
						return nil
					}
					return []reconcile.Request{
						{
							NamespacedName: types.NamespacedName{
								Name: na.Name,
							},
						},
					}
				}),
			),
		).
		Complete(r)
}
