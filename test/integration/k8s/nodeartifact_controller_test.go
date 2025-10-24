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

package k8s

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
)

// nolint:dupl
var _ = Describe("NodeArtifact Controller", func() {
	ctx := context.Background()
	nodeName := "node-1"
	timeout := 10 * time.Second
	interval := 1 * time.Second

	Context("When reconciling a new resource", func() {
		It("should successfully create the resource", func() {
			resourceName := "test-new-resource"
			typeNamespacedName := types.NamespacedName{
				Name: resourceName,
			}

			By("creating the custom resource for the Kind NodeArtifact")
			resource := &metalk8sv1alpha1.NodeArtifact{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeArtifactSpec{
					Name:     "solution-1",
					Version:  "1.2.0",
					NodeName: nodeName,
					Validation: metalk8sv1alpha1.ArtifactValidation{
						Checksum: metalk8sv1alpha1.NodeArtifactChecksum{
							Type:  "sha256",
							Value: "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
						},
					},
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeArtifact")
			createdResource := &metalk8sv1alpha1.NodeArtifact{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Available).To(BeFalse())
		})
	})

	Context("When reconciling a resource with available artifact", func() {
		It("should successfully create the resource", func() {
			resourceName := "test-available-resource"
			typeNamespacedName := types.NamespacedName{
				Name: resourceName,
			}

			By("creating the custom resource for the Kind NodeArtifact")
			resource := &metalk8sv1alpha1.NodeArtifact{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeArtifactSpec{
					Name:     "solution-2",
					Version:  "4.2.1",
					NodeName: nodeName,
					Validation: metalk8sv1alpha1.ArtifactValidation{
						Checksum: metalk8sv1alpha1.NodeArtifactChecksum{
							Type:  "sha256",
							Value: "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
						},
					},
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeArtifact")
			createdResource := &metalk8sv1alpha1.NodeArtifact{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Available).To(BeTrue())
		})
	})

	Context("When reconciling a resource with invalid artifact", func() {
		It("should successfully create the resource", func() {
			resourceName := "test-invalid-resource"
			typeNamespacedName := types.NamespacedName{
				Name: resourceName,
			}

			By("creating the custom resource for the Kind NodeArtifact")
			resource := &metalk8sv1alpha1.NodeArtifact{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeArtifactSpec{
					Name:     "solution-2",
					Version:  "4.2.1",
					NodeName: nodeName,
					Validation: metalk8sv1alpha1.ArtifactValidation{
						Checksum: metalk8sv1alpha1.NodeArtifactChecksum{
							Type:  "sha256",
							Value: "bad",
						},
					},
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeArtifact")
			createdResource := &metalk8sv1alpha1.NodeArtifact{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Available).To(BeFalse())
		})
	})
})
