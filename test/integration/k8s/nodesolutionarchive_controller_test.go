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

//nolint:goconst
package k8s

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"github.com/scality/metalk8s-registry-node-agent/test/utils"
)

func archiveState(name, version, state string) func() float64 {
	return func() float64 {
		value, _ := utils.MetricValue("registry_nsa_state", map[string]string{
			"name": name, "version": version, "state": state,
		})
		return value
	}
}

func archiveStateExists(name, version string) func() bool {
	return func() bool {
		_, exists := utils.MetricValue("registry_nsa_state", map[string]string{
			"name": name, "version": version, "state": "initialized",
		})
		return exists
	}
}

// nolint:dupl
var _ = Describe("NodeSolutionArchive Controller", func() {
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

			By("creating the custom resource for the Kind NodeSolutionArchive")
			resource := &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "solution-1",
						Version: "1.2.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
							},
						},
					},
					NodeName: nodeName,
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeSolutionArchive")
			createdResource := &metalk8sv1alpha1.NodeSolutionArchive{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Initialized).To(BeTrue())
			Expect(*createdResource.Status.Available).To(BeFalse())
			Expect(*createdResource.Status.Served).To(BeFalse())

			By("exposing the state of the resource as metrics")
			Eventually(archiveState("solution-1", "1.2.0", "initialized"), timeout, interval).Should(Equal(1.0))
			Expect(archiveState("solution-1", "1.2.0", "available")()).To(Equal(0.0))
			Expect(archiveState("solution-1", "1.2.0", "served")()).To(Equal(0.0))
		})
	})

	Context("When reconciling a resource with available solution archive", func() {
		It("should successfully create the resource", func() {
			resourceName := "test-available-resource"
			typeNamespacedName := types.NamespacedName{
				Name: resourceName,
			}

			By("creating the custom resource for the Kind NodeSolutionArchive")
			resource := &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "solution-2",
						Version: "4.2.1",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "ce775a33b30ae640d521df1fad60868fa701707ffdc4d8b4ca7ab60edfd05c26",
							},
						},
					},
					NodeName: nodeName,
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeSolutionArchive")
			createdResource := &metalk8sv1alpha1.NodeSolutionArchive{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Initialized).To(BeTrue())
			Expect(*createdResource.Status.Available).To(BeTrue())
			Expect(*createdResource.Status.Served).To(BeTrue())

			By("exposing the state of the resource as metrics")
			Eventually(archiveState("solution-2", "4.2.1", "served"), timeout, interval).Should(Equal(1.0))
			Expect(archiveState("solution-2", "4.2.1", "initialized")()).To(Equal(1.0))
			Expect(archiveState("solution-2", "4.2.1", "available")()).To(Equal(1.0))
		})
	})

	Context("When reconciling a resource with invalid solution archive", func() {
		It("should successfully create the resource", func() {
			resourceName := "test-invalid-resource"
			typeNamespacedName := types.NamespacedName{
				Name: resourceName,
			}

			By("creating the custom resource for the Kind NodeSolutionArchive")
			resource := &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "solution-2",
						Version: "4.2.1",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "bad",
							},
						},
					},
					NodeName: nodeName,
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeSolutionArchive")
			createdResource := &metalk8sv1alpha1.NodeSolutionArchive{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Initialized).To(BeTrue())
			Expect(*createdResource.Status.Available).To(BeFalse())
			Expect(*createdResource.Status.Served).To(BeFalse())

			By("counting the archive checksum failure")
			checksumFailures, _ := utils.MetricValue("registry_nsa_integrity_failures_total", map[string]string{
				"name": "solution-2", "version": "4.2.1", "stage": "archive_checksum",
			})
			Expect(checksumFailures).To(BeNumerically(">=", 1))
		})
	})

	Context("When reconciling a resource with external solution archive", func() {
		It("should successfully create the resource", func() {
			resourceNameNode1 := "test-external-resource-node1"
			typeNamespacedNameNode1 := types.NamespacedName{
				Name: resourceNameNode1,
			}

			resourceNameNode2 := "test-external-resource-node2"

			By("creating a custom resource for the Kind NodeSolutionArchive on other Node")
			otherResource := &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceNameNode2,
				},
			}
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, otherResource, func() error {
				otherResource.Spec = metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "solution-3",
						Version: "4.2.1",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "95162a9fe88f9d11c7f7ef7dc20c2426814e188fd858b27adb5274d3689675af",
							},
						},
					},
					NodeName: "otherNodeName",
				}

				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			By("updating its status to initialized and available")
			otherResource.Status = metalk8sv1alpha1.NodeSolutionArchiveStatus{
				Initialized: ptr.To(true),
				Available:   ptr.To(true),
				URL:         "https://example.com:5002/api/v1/downloads/solution-3/4.2.1",
			}
			err = k8sClient.Status().Update(ctx, otherResource)

			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind NodeSolutionArchive")
			resource := &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceNameNode1,
				},
			}

			_, err = controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "solution-3",
						Version: "4.2.1",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "95162a9fe88f9d11c7f7ef7dc20c2426814e188fd858b27adb5274d3689675af",
							},
						},
					},
					NodeName: nodeName,
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeSolutionArchive")
			createdResource := &metalk8sv1alpha1.NodeSolutionArchive{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedNameNode1, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Initialized).To(BeTrue())
			Expect(*createdResource.Status.Available).To(BeTrue())
			Expect(*createdResource.Status.Served).To(BeTrue())
		})
	})

	Context("When reconciling a resource with error during initialization", func() {
		It("should fail to initialize the resource", func() {
			resourceName := "test-not-initialized-resource"
			typeNamespacedName := types.NamespacedName{
				Name: resourceName,
			}

			By("creating the custom resource for the Kind NodeSolutionArchive")
			resource := &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, resource, func() error {
				resource.Spec = metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "solution-4",
						Version: "4.2.8",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "a51ae4b357df473f4f5c84fde00e22c66206c5d41a0c9d8f2cadd4e5cb6d06c4",
							},
						},
					},
					NodeName: nodeName,
				}
				return nil
			})

			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("checking the custom resource for the Kind NodeSolutionArchive")
			createdResource := &metalk8sv1alpha1.NodeSolutionArchive{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespacedName, createdResource)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdResource.Spec).To(Equal(resource.Spec))
			Expect(*createdResource.Status.Initialized).To(BeFalse())
			Eventually(archiveStateExists("solution-4", "4.2.8"), timeout, interval).Should(BeTrue())
			Expect(archiveState("solution-4", "4.2.8", "initialized")()).To(Equal(0.0))

			By("deleting the NodeSolutionArchive")
			err = k8sClient.Delete(ctx, resource)
			Expect(err).NotTo(HaveOccurred())

			// Wait for all reconciliations loop to be done
			time.Sleep(1 * time.Second)

			By("dropping the state metrics of the deleted resource")
			Eventually(archiveStateExists("solution-4", "4.2.8"), timeout, interval).Should(BeFalse())
		})
	})
})
