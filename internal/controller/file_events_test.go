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
package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

// fakeClient is a fake Kubernetes client for testing
type fakeClient struct {
	mu        sync.RWMutex
	resources map[string]*metalk8sv1alpha1.NodeSolutionArchive
}

// newFakeClient creates a new fake client
func newFakeClient() *fakeClient {
	return &fakeClient{
		resources: make(map[string]*metalk8sv1alpha1.NodeSolutionArchive),
	}
}

// List retrieves a list of objects
func (f *fakeClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	nsaList, ok := list.(*metalk8sv1alpha1.NodeSolutionArchiveList)
	if !ok {
		return fmt.Errorf("unsupported list type")
	}

	// Parse list options to extract matching fields
	listOpts := &client.ListOptions{}
	for _, opt := range opts {
		opt.ApplyToList(listOpts)
	}

	// Kubernetes connection failure testing
	if listOpts.FieldSelector != nil &&
		(listOpts.FieldSelector.Matches(fields.Set{"LocalSolutionArchiveName": "test-kubernetes-connection-failure"}) ||
			listOpts.FieldSelector.Matches(fields.Set{"LocalSolutionArchiveNameVersion": "test-kubernetes-connection-failure-1.0.0"})) {
		return errors.New("kubernetes connection failure error")
	}

	// Filter resources based on matching fields
	var filtered []metalk8sv1alpha1.NodeSolutionArchive
	for _, resource := range f.resources {
		if f.matchesFields(resource, listOpts.FieldSelector) {
			filtered = append(filtered, *resource.DeepCopy())
		}
	}

	nsaList.Items = filtered
	return nil
}

// matchesFields checks if a resource matches the field selector
func (f *fakeClient) matchesFields(resource *metalk8sv1alpha1.NodeSolutionArchive, selector fields.Selector) bool {
	if selector == nil {
		return true
	}

	// Extract the matching fields from the selector
	// In our case, we're looking for "LocalSolutionArchiveNameVersion"
	// The selector string format is "field=value"
	requirements := selector.Requirements()
	for _, req := range requirements {
		if req.Field == "LocalSolutionArchiveNameVersion" {
			// Construct the name-version string from the resource
			resourceNameVersion := fmt.Sprintf("%s-%s", resource.Spec.Name, resource.Spec.Version)
			return resourceNameVersion == req.Value
		}
		if req.Field == "LocalSolutionArchiveName" {
			return resource.Spec.Name == req.Value
		}
	}

	return true
}

// Create creates an object
func (f *fakeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	nsa, ok := obj.(*metalk8sv1alpha1.NodeSolutionArchive)
	if !ok {
		return fmt.Errorf("unsupported object type")
	}

	f.resources[nsa.Name] = nsa.DeepCopy()
	return nil
}

// Delete deletes an object
func (f *fakeClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	nsa, ok := obj.(*metalk8sv1alpha1.NodeSolutionArchive)
	if !ok {
		return fmt.Errorf("unsupported object type")
	}

	// Actually delete the resource from the map
	if _, exists := f.resources[nsa.Name]; exists {
		delete(f.resources, nsa.Name)
		return nil
	}

	return fmt.Errorf("not found")
}

var _ = Describe("FileEvents", func() {
	var (
		fileEventHandler *FileEvents
		filenameCh       chan domain.FileEventDetails
		reconcileCh      chan event.GenericEvent
		deleteCh         chan domain.FileEventDetails
		testCtx          context.Context
		testCancel       context.CancelFunc
		logger           zerolog.Logger
		k8sClient        *fakeClient
	)

	BeforeEach(func() {
		// Set up test context
		testCtx, testCancel = context.WithCancel(context.Background())

		// Set up channels
		filenameCh = make(chan domain.FileEventDetails)
		reconcileCh = make(chan event.GenericEvent)
		deleteCh = make(chan domain.FileEventDetails)

		// Set up logger
		logger = zerolog.Nop()

		// Create fake client
		k8sClient = newFakeClient()

		// Create FileEvents instance
		fileEventHandler = NewFileEvents(
			testCtx,
			&logger,
			k8sClient,
			filenameCh,
			reconcileCh,
			deleteCh,
		)
	})

	AfterEach(func() {
		// Clean up
		testCancel()
		close(filenameCh)
		close(reconcileCh)
		close(deleteCh)
	})

	Context("When processing SolutionArchivesOrigin events", func() {
		var testResource *metalk8sv1alpha1.NodeSolutionArchive
		var underDeletionTestResource *metalk8sv1alpha1.NodeSolutionArchive

		BeforeEach(func() {
			// Create a test NodeSolutionArchive resource
			testResource = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-solution-archive",
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution",
						Version: "1.0.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "abc123",
							},
						},
					},
					NodeName: "test-node",
				},
			}
			Expect(k8sClient.Create(testCtx, testResource)).To(Succeed())

			// Create an underDeletion test NodeSolutionArchive resource
			underDeletionTestResource = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-solution-archive-under-deletion",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution-under-deletion",
						Version: "1.0.1",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "abc123",
							},
						},
					},
					NodeName: "test-node",
				},
			}
			Expect(k8sClient.Create(testCtx, underDeletionTestResource)).To(Succeed())
		})

		AfterEach(func() {
			// Clean up the test resource
			Expect(k8sClient.Delete(testCtx, testResource)).To(Succeed())
			Expect(k8sClient.Delete(testCtx, underDeletionTestResource)).To(Succeed())
		})

		/*
			Create Event Tests
		*/
		It("should trigger reconcile on Create event for a file when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Create event for a file with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-1.0.0.iso",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that a reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Create event for a file when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Create event for a file with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-under-deletion-1.0.1.iso",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger delete on Create event for a file when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Create event for a file with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-non-existent-1.0.2.iso",
				ObjectName:   "test-solution-non-existent-1.0.2",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should neither trigger reconcile nor delete on Create event for a directory when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Create event for a directory with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-1.0.0",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger delete on Create event for a directory when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Create event for a directory with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-under-deletion-1.0.1",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on Create event for a directory when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Create event for a directory with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-non-existent-1.0.2",
				ObjectName:   "test-solution-non-existent-1.0.2",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verifu no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		/*
			Remove Event Tests
		*/

		It("should trigger reconcile on Remove event for a file when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Remove event for a file with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-1.0.0.iso",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that a reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a file when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Remove event for a file with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-under-deletion-1.0.1.iso",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a file when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Remove event for a file with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/nonexistent-solution-2.0.0.iso",
				ObjectName:   "nonexistent-solution-2.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Remove event for a directory when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Remove event for a directory with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-1.0.0",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that a reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Remove event for a directory with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-under-deletion-1.0.1",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Remove event for a directory with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/nonexistent-solution-2.0.0.iso",
				ObjectName:   "nonexistent-solution-2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		/*
			Write Event Tests
		*/
		It("should trigger reconcile on Write event for a file when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Write event for a file with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-1.0.0.iso",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that a reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a file when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Write event for a file with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-under-deletion-1.0.1.iso",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger delete on Write event for a file when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Write event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-non-existent-1.0.2.iso",
				ObjectName:   "test-solution-non-existent-1.0.2",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a directory when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Write event for a directory with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-1.0.0",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger delete on Write event for a directory when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Write event for a file with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-under-deletion-1.0.1",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on Write event for a directory when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Write event for a directory with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-non-existent-1.0.2",
				ObjectName:   "test-solution-non-existent-1.0.2",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		/*
			Rename Event Tests
		*/
		It("should trigger reconcile on Rename event for a file when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Rename event for a file with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-1.0.0.iso",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that a reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Rename event for a file when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Rename event for a file with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-solution-under-deletion-1.0.1.iso",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Rename event for a file when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Rename event for a file with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/nonexistent-solution-2.0.0.iso",
				ObjectName:   "nonexistent-solution-2.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Rename event for a directory when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Rename event for a directory with existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-1.0.0",
				ObjectName:   "test-solution-1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that a reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Rename event for a directory when CR exists and is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Rename event for a directory with under deletion existing CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.test-solution-under-deletion-1.0.1",
				ObjectName:   "test-solution-under-deletion-1.0.1",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Rename event for a directory when CR does not exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send a Rename event for a directory with no matching CR
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/.bucket.nonexistent-solution-2.0.0",
				ObjectName:   "nonexistent-solution-2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should requeue a processing event when kubernetes connection fails", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/archives/test-kubernetes-connection-failure-1.0.0.iso",
				ObjectName:   "test-kubernetes-connection-failure-1.0.0",
				IsDir:        false,
				Origin:       domain.SolutionArchivesOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that the event was requeued on filenameChan
			Eventually(filenameCh, time.Second*5).Should(Receive())
			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})
	})

	Context("When processing SolutionsOrigin events", func() {
		var underDeletionTestResource1 *metalk8sv1alpha1.NodeSolutionArchive
		var underDeletionTestResource2 *metalk8sv1alpha1.NodeSolutionArchive
		var underDeletionMixedTestResource *metalk8sv1alpha1.NodeSolutionArchive
		var validMixedTestResource1 *metalk8sv1alpha1.NodeSolutionArchive
		var validMixedTestResource2 *metalk8sv1alpha1.NodeSolutionArchive

		BeforeEach(func() {
			// Create a test NodeSolutionArchive resource
			underDeletionTestResource1 = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-solution-archive-under-deletion1",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution-under-deletion",
						Version: "1.0.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "def456",
							},
						},
					},
					NodeName: "test-node",
				},
				Status: metalk8sv1alpha1.NodeSolutionArchiveStatus{
					Available: ptr.To(true),
				},
			}
			Expect(k8sClient.Create(testCtx, underDeletionTestResource1)).To(Succeed())

			// Create an underDeletion test NodeSolutionArchive resource
			underDeletionTestResource2 = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-solution-archive-under-deletion2",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution-under-deletion",
						Version: "2.0.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "abc123",
							},
						},
					},
					NodeName: "test-node",
				},
				Status: metalk8sv1alpha1.NodeSolutionArchiveStatus{
					Available: ptr.To(true),
				},
			}
			Expect(k8sClient.Create(testCtx, underDeletionTestResource2)).To(Succeed())

			// Create a valid test NodeSolutionArchive resource
			validMixedTestResource1 = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-solution-mixed-valid1",
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution-mixed",
						Version: "1.0.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "abc123",
							},
						},
					},
					NodeName: "test-node",
				},
				Status: metalk8sv1alpha1.NodeSolutionArchiveStatus{
					Available: ptr.To(true),
				},
			}
			Expect(k8sClient.Create(testCtx, validMixedTestResource1)).To(Succeed())

			// Create an underDeletion test NodeSolutionArchive resource
			underDeletionMixedTestResource = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-solution-mixed-under-deletion",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution-mixed",
						Version: "2.0.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "def456",
							},
						},
					},
					NodeName: "test-node",
				},
				Status: metalk8sv1alpha1.NodeSolutionArchiveStatus{
					Available: ptr.To(true),
				},
			}
			Expect(k8sClient.Create(testCtx, underDeletionMixedTestResource)).To(Succeed())

			// Create a valid test NodeSolutionArchive resource
			validMixedTestResource2 = &metalk8sv1alpha1.NodeSolutionArchive{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-solution-mixed-valid2",
				},
				Spec: metalk8sv1alpha1.NodeSolutionArchiveSpec{
					SolutionArchiveSpec: metalk8sv1alpha1.SolutionArchiveSpec{
						Name:    "test-solution-mixed",
						Version: "3.0.0",
						Validation: &metalk8sv1alpha1.SolutionArchiveValidation{
							Checksum: metalk8sv1alpha1.SolutionArchiveChecksum{
								Type:  "sha256",
								Value: "abc123",
							},
						},
					},
					NodeName: "test-node",
				},
				Status: metalk8sv1alpha1.NodeSolutionArchiveStatus{
					Available: ptr.To(true),
				},
			}
			Expect(k8sClient.Create(testCtx, validMixedTestResource2)).To(Succeed())
		})

		AfterEach(func() {
			// Clean up the test resource
			Expect(k8sClient.Delete(testCtx, underDeletionTestResource1)).To(Succeed())
			Expect(k8sClient.Delete(testCtx, underDeletionTestResource2)).To(Succeed())
			Expect(k8sClient.Delete(testCtx, validMixedTestResource1)).To(Succeed())
			Expect(k8sClient.Delete(testCtx, validMixedTestResource2)).To(Succeed())
			Expect(k8sClient.Delete(testCtx, underDeletionMixedTestResource)).To(Succeed())
		})

		/*
			Create Event Tests
		*/
		It("should trigger delete on Create event for a file directly in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/some-file.txt",
				ObjectName:   "some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on Create event for a file in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution/some-file.txt",
				ObjectName:   "test-solution/some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should neither trigger reconcile nor delete on Create event for a directory in root solutions when at least one CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed",
				ObjectName:   "test-solution-mixed",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Create event for a directory in root solutions when all CRs are under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-under-deletion",
				ObjectName:   "test-solution-under-deletion",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger delete on Create event for a directory in root solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent",
				ObjectName:   "test-solution-non-existent",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should neither trigger reconcile nor delete on Create event for a directory in solutions when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/1.0.0",
				ObjectName:   "test-solution-mixed/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Create event for a directory in solutions when CR is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/2.0.0",
				ObjectName:   "test-solution-mixed/2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger delete on Create event for a directory in solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent/1.0.0",
				ObjectName:   "test-solution-non-existent/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		/*
			Remove Event Tests
		*/
		It("should trigger delete on Remove event for a file directly in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/some-file.txt",
				ObjectName:   "some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on Remove event for a file in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution/some-file.txt",
				ObjectName:   "test-solution/some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger reconcile on Remove event for a directory in root solutions when at least one CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed",
				ObjectName:   "test-solution-mixed",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that 3 reconcile events were triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no additional reconcile events are triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Remove event for a directory in root solutions when all CRs are under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-under-deletion",
				ObjectName:   "test-solution-under-deletion",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that 2 reconcile events were triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no additional reconcile events are triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory in root solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent",
				ObjectName:   "test-solution-non-existent",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Remove event for a directory in solutions when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/1.0.0",
				ObjectName:   "test-solution-mixed/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory in solutions when CR is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/2.0.0",
				ObjectName:   "test-solution-mixed/2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory in solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent/1.0.0",
				ObjectName:   "test-solution-non-existent/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		/*
			Write Event Tests
		*/
		It("should trigger delete on Write event for a file directly in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/some-file.txt",
				ObjectName:   "some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on Write event for a file in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution/some-file.txt",
				ObjectName:   "test-solution/some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a directory in root solutions when at least one CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed",
				ObjectName:   "test-solution-mixed",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a directory in root solutions when all CRs are under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-under-deletion",
				ObjectName:   "test-solution-under-deletion",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a directory in root solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent",
				ObjectName:   "test-solution-non-existent",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Write event for a directory in solutions when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/1.0.0",
				ObjectName:   "test-solution-mixed/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a directory in solutions when CR is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/2.0.0",
				ObjectName:   "test-solution-mixed/2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Write event for a directory in solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent/1.0.0",
				ObjectName:   "test-solution-non-existent/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Write.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		/*
			Rename Event Tests
		*/
		It("should trigger delete on Rename event for a file directly in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/some-file.txt",
				ObjectName:   "some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on Rename event for a file in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution/some-file.txt",
				ObjectName:   "test-solution/some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger reconcile on Rename event for a directory in root solutions when at least one CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed",
				ObjectName:   "test-solution-mixed",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that 3 reconcile events were triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no additional reconcile events are triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Rename event for a directory in root solutions when all CRs are under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-under-deletion",
				ObjectName:   "test-solution-under-deletion",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that 2 reconcile events were triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no additional reconcile events are triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Rename event for a directory in root solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent",
				ObjectName:   "test-solution-non-existent",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Rename.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on Remove event for a directory in solutions when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/1.0.0",
				ObjectName:   "test-solution-mixed/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory in solutions when CR is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/2.0.0",
				ObjectName:   "test-solution-mixed/2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on Remove event for a directory in solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent/1.0.0",
				ObjectName:   "test-solution-non-existent/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Remove.String(),
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		/*
			[no events] Event Tests
			Observed when unmounting a filesystem (e.g. when a solution is unmounted)
		*/
		It("should trigger delete on [no events] event for a file directly in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/some-file.txt",
				ObjectName:   "some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should trigger delete on [no events] event for a file in solutions directory", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a file in the solutions directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution/some-file.txt",
				ObjectName:   "test-solution/some-file.txt",
				IsDir:        false,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that a delete event was triggered
			Eventually(deleteCh, time.Second*5).Should(Receive())
		})

		It("should neither trigger reconcile nor delete on [no events] event for a directory in root solutions when at least one CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed",
				ObjectName:   "test-solution-mixed",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on [no events] event for a directory in root solutions when all CRs are under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-under-deletion",
				ObjectName:   "test-solution-under-deletion",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on [no events] event for a directory in root solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent",
				ObjectName:   "test-solution-non-existent",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should trigger reconcile on [no events] event for a directory in solutions when CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/1.0.0",
				ObjectName:   "test-solution-mixed/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Eventually(reconcileCh, time.Second*5).Should(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on [no events] event for a directory in solutions when CR is under deletion", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-mixed/2.0.0",
				ObjectName:   "test-solution-mixed/2.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should neither trigger reconcile nor delete on [no events] event for a directory in solutions when no CR exists", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-solution-non-existent/1.0.0",
				ObjectName:   "test-solution-non-existent/1.0.0",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    "[no events]",
			}

			// Verify that no reconcile event was triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			// Verify that no delete event was triggered
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})

		It("should requeue a processing event when kubernetes connection fails", func() {
			// Start listening in a goroutine
			go fileEventHandler.Listen()

			// Send an event for a directory
			filenameCh <- domain.FileEventDetails{
				FullPathName: "/solutions/test-kubernetes-connection-failure",
				ObjectName:   "test-kubernetes-connection-failure",
				IsDir:        true,
				Origin:       domain.SolutionsOrigin,
				EventType:    fsnotify.Create.String(),
			}

			// Verify that the event was requeued on filenameChan
			Eventually(filenameCh, time.Second*5).Should(Receive())
			// Verify that no reconcile/delete were triggered
			Consistently(reconcileCh, time.Millisecond*500).ShouldNot(Receive())
			Consistently(deleteCh, time.Millisecond*500).ShouldNot(Receive())
		})
	})
})
