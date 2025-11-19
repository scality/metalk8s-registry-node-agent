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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/hashicorp/go-version"
	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var nodesolutionarchivelog = logf.Log.WithName("nodesolutionarchive-resource")

// SetupNodeSolutionArchiveWebhookWithManager registers the webhook for NodeSolutionArchive in the manager.
func SetupNodeSolutionArchiveWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&metalk8sv1alpha1.NodeSolutionArchive{}).
		WithValidator(&NodeSolutionArchiveCustomValidator{
			client: mgr.GetClient(),
		}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-metalk8s-scality-com-v1alpha1-nodesolutionarchive,mutating=false,failurePolicy=fail,sideEffects=None,groups=metalk8s.scality.com,resources=nodesolutionarchives,verbs=create,versions=v1alpha1,name=vnodesolutionarchive-v1alpha1.kb.io,admissionReviewVersions=v1

// NodeSolutionArchiveCustomValidator struct is responsible for validating the NodeSolutionArchive resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NodeSolutionArchiveCustomValidator struct {
	client client.Client
}

var _ webhook.CustomValidator = &NodeSolutionArchiveCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type NodeSolutionArchive.
func (v *NodeSolutionArchiveCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	nodesolutionarchive, ok := obj.(*metalk8sv1alpha1.NodeSolutionArchive)
	if !ok {
		return nil, fmt.Errorf("expected a NodeSolutionArchive object but got %T", obj)
	}
	nodesolutionarchivelog.Info("Validation for NodeSolutionArchive upon creation", "name", nodesolutionarchive.GetName())

	return nil, validateNodeSolutionArchive(ctx, v.client, nodesolutionarchive)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type NodeSolutionArchive.
func (v *NodeSolutionArchiveCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	// Careful: not activated by default
	// To enable it think about changing "verbs=create" to "verbs=create,update" in "+kubebuilder:webhook" annotation above
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type NodeSolutionArchive.
func (v *NodeSolutionArchiveCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// Careful: not activated by default
	// To enable it think about changing "verbs=create" to "verbs=create,delete" in "+kubebuilder:webhook" annotation above
	return nil, nil
}

func validateNodeSolutionArchive(ctx context.Context, c client.Client, nodesolutionarchive *metalk8sv1alpha1.NodeSolutionArchive) error {
	// Validate the version is conform to SemVer convention
	_, err := version.NewVersion(nodesolutionarchive.Spec.Version)
	if err != nil {
		return fmt.Errorf("version is not conform to SemVer convention: %s", nodesolutionarchive.Spec.Version)
	}
	artifactNameVersion := fmt.Sprintf("%s-%s", nodesolutionarchive.Spec.Name, nodesolutionarchive.Spec.Version)
	naList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
	if err := c.List(ctx, naList, client.MatchingFields{"SolutionArchiveNameVersion": artifactNameVersion}); err != nil {
		return err
	}
	for _, item := range naList.Items {
		if item.Spec.NodeName == nodesolutionarchive.Spec.NodeName {
			return fmt.Errorf("artifact %s already exists", artifactNameVersion)
		}
	}

	return nil
}
