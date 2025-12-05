package controller

import (
	"context"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	metalk8sv1alpha1 "github.com/scality/metalk8s-registry-node-agent/api/v1alpha1"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type KubernetesClientInterface interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	// Create and Delete are used for testing purposes
	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

type FileEvents struct {
	ctx           context.Context
	logger        *zerolog.Logger
	client        KubernetesClientInterface
	filenameChan  chan domain.FileEventDetails
	reconcileChan chan event.GenericEvent
	deleteChan    chan domain.FileEventDetails
}

func NewFileEvents(
	ctx context.Context,
	l *zerolog.Logger,
	c KubernetesClientInterface,
	filenameChan chan domain.FileEventDetails,
	reconcileChan chan event.GenericEvent,
	deleteChan chan domain.FileEventDetails,
) *FileEvents {
	return &FileEvents{
		ctx:           ctx,
		logger:        l,
		client:        c,
		filenameChan:  filenameChan,
		reconcileChan: reconcileChan,
		deleteChan:    deleteChan,
	}
}

// Listen runs in a goroutine, processing FileEventDetails from a channel
func (f *FileEvents) Listen() {
	var err error
	for eventDetails := range f.filenameChan {
		switch eventDetails.Origin {
		case domain.SolutionArchivesOrigin:
			err = f.handleSolutionArchiveEvent(eventDetails)
			if err != nil {
				f.logger.Error().Err(err).Msg("Failed to handle solution archive event")
				f.filenameChan <- eventDetails
			}

		case domain.SolutionsOrigin:
			err = f.handleSolutionEvent(eventDetails)
			if err != nil {
				f.logger.Error().Err(err).Msg("Failed to handle solution event")
				f.filenameChan <- eventDetails
			}
		}
	}
}

// handleSolutionArchiveEvent processes events from the solution archives directory
func (f *FileEvents) handleSolutionArchiveEvent(eventDetails domain.FileEventDetails) error {
	foundCR, err := f.findSolutionArchiveCR(eventDetails.ObjectName)
	if err != nil {
		f.logger.Error().Err(err).Msg("Failed to contact kubernetes cluster")
		return err
	}

	var underDeletion bool
	if foundCR != nil {
		underDeletion = !foundCR.DeletionTimestamp.IsZero()
	}

	switch eventDetails.EventType {
	case fsnotify.Create.String():
		f.handleArchiveCreate(eventDetails, foundCR, underDeletion)
	case fsnotify.Remove.String(), fsnotify.Rename.String():
		f.handleArchiveRemoveOrRename(foundCR, underDeletion)
	case fsnotify.Write.String():
		f.handleArchiveWrite(eventDetails, foundCR, underDeletion)
	}
	return nil
}

// handleSolutionEvent processes events from the solutions directory
func (f *FileEvents) handleSolutionEvent(eventDetails domain.FileEventDetails) error {
	// Delete any files in the solutions directory
	if !eventDetails.IsDir {
		f.deleteChan <- eventDetails
		return nil
	}

	objectIsVersioned := strings.Contains(eventDetails.ObjectName, "/")
	nsaList, err := f.findSolutionCRs(eventDetails.ObjectName, objectIsVersioned)
	if err != nil {
		f.logger.Error().Err(err).Msg("Failed to contact kubernetes cluster")
		return err
	}

	if nsaList == nil {
		return nil
	}

	switch eventDetails.EventType {
	case fsnotify.Create.String():
		f.handleSolutionCreate(eventDetails, nsaList)
	case fsnotify.Remove.String(), fsnotify.Rename.String():
		f.handleSolutionRemoveOrRename(nsaList, objectIsVersioned)
	case fsnotify.Write.String():
		f.handleSolutionWrite(nsaList, objectIsVersioned)
	case fsnotify.Chmod.String():
		return nil
	// When mounting/unmounting, a [no events] event is raised
	default:
		f.handleSolutionDefault(nsaList, objectIsVersioned)
	}
	return nil
}

// findSolutionArchiveCR finds a CR matching the solution archive name-version
func (f *FileEvents) findSolutionArchiveCR(objectName string) (*metalk8sv1alpha1.NodeSolutionArchive, error) {
	nsaList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
	err := f.client.List(f.ctx, nsaList, client.MatchingFields{"LocalSolutionArchiveNameVersion": objectName})
	if err != nil {
		f.logger.Error().Err(err).Msg("Failed to list custom resources")
		return nil, err
	}

	if len(nsaList.Items) > 0 {
		return &nsaList.Items[0], nil
	}
	return nil, nil
}

// findSolutionCRs finds CRs matching the solution name or name-version
func (f *FileEvents) findSolutionCRs(objectName string, isVersioned bool) (*metalk8sv1alpha1.NodeSolutionArchiveList, error) {
	nsaList := &metalk8sv1alpha1.NodeSolutionArchiveList{}
	var err error

	if isVersioned {
		// Convert <solution>/<version> to <solution>-<version>
		parts := strings.Split(objectName, "/")
		objectNameVersion := parts[0] + "-" + parts[1]
		err = f.client.List(f.ctx, nsaList, client.MatchingFields{"LocalSolutionArchiveNameVersion": objectNameVersion})
	} else {
		err = f.client.List(f.ctx, nsaList, client.MatchingFields{"LocalSolutionArchiveName": objectName})
	}

	if err != nil {
		f.logger.Error().Err(err).Msg("Failed to list custom resources")
		return nil, err
	}
	return nsaList, nil
}

// handleArchiveCreate handles Create events for solution archives
// Files: Reconcile if CR exists and not under deletion, delete if no CR exists, do nothing if under deletion
// Directories: Delete if CR doesn't exist or is under deletion
func (f *FileEvents) handleArchiveCreate(eventDetails domain.FileEventDetails, foundCR *metalk8sv1alpha1.NodeSolutionArchive, underDeletion bool) {
	if foundCR != nil && !underDeletion && !eventDetails.IsDir {
		f.queueReconcile(foundCR)
	} else if foundCR == nil || (underDeletion && eventDetails.IsDir) {
		f.deleteChan <- eventDetails
	}
}

// handleArchiveRemoveOrRename handles Remove/Rename events for solution archives
// Reconcile if CR exists and not under deletion to trigger download or bucket recreation
func (f *FileEvents) handleArchiveRemoveOrRename(foundCR *metalk8sv1alpha1.NodeSolutionArchive, underDeletion bool) {
	if foundCR != nil && !underDeletion {
		f.queueReconcile(foundCR)
	}
}

// handleArchiveWrite handles Write events for solution archives
// Files: Reconcile if CR exists and not under deletion to check checksum
// Directories: Delete if CR doesn't exist or is under deletion
func (f *FileEvents) handleArchiveWrite(eventDetails domain.FileEventDetails, foundCR *metalk8sv1alpha1.NodeSolutionArchive, underDeletion bool) {
	if foundCR != nil && !underDeletion && !eventDetails.IsDir {
		f.queueReconcile(foundCR)
	} else if foundCR == nil || (underDeletion && eventDetails.IsDir) {
		f.deleteChan <- eventDetails
	}
}

// handleSolutionCreate handles Create events for solutions directory
// Versioned paths: Do nothing if CR exists, delete if no CR exists
// Non-versioned paths: Do nothing (root directory)
// No CRs found: Delete the directory
func (f *FileEvents) handleSolutionCreate(eventDetails domain.FileEventDetails, nsaList *metalk8sv1alpha1.NodeSolutionArchiveList) {
	if len(nsaList.Items) == 0 {
		f.deleteChan <- eventDetails
		return
	}
	// For both versioned and non-versioned paths, do nothing if CRs exist
}

// handleSolutionRemoveOrRename handles Remove/Rename events for solutions directory
// Root paths: Reconcile all CRs
// Versioned paths: Reconcile the CR if not under deletion
func (f *FileEvents) handleSolutionRemoveOrRename(nsaList *metalk8sv1alpha1.NodeSolutionArchiveList, objectIsVersioned bool) {
	if len(nsaList.Items) == 0 {
		return
	}

	if objectIsVersioned {
		// For versioned paths, only reconcile CRs not under deletion
		for _, nsa := range nsaList.Items {
			if nsa.DeletionTimestamp.IsZero() {
				f.queueReconcile(&nsa)
				return
			}
		}
	} else {
		// For root paths, reconcile all CRs
		for i := range nsaList.Items {
			f.queueReconcile(&nsaList.Items[i])
		}
	}
}

// handleSolutionWrite handles Write events for solutions directory
// Root paths: Do nothing
// Versioned paths: Reconcile the CR if it exists and not under deletion
func (f *FileEvents) handleSolutionWrite(nsaList *metalk8sv1alpha1.NodeSolutionArchiveList, objectIsVersioned bool) {
	if len(nsaList.Items) == 0 || !objectIsVersioned {
		return
	}

	// For versioned paths, reconcile the CR if not under deletion
	for _, nsa := range nsaList.Items {
		if nsa.DeletionTimestamp.IsZero() {
			f.queueReconcile(&nsa)
			return
		}
	}
}

// handleSolutionDefault handles [no events] events for solutions directory
// Root paths: Do nothing
// Versioned paths: Reconcile the CR if it exists and not under deletion
func (f *FileEvents) handleSolutionDefault(nsaList *metalk8sv1alpha1.NodeSolutionArchiveList, objectIsVersioned bool) {
	if len(nsaList.Items) == 0 || !objectIsVersioned {
		return
	}

	// For versioned paths, reconcile the CR if not under deletion
	for _, nsa := range nsaList.Items {
		if nsa.DeletionTimestamp.IsZero() {
			f.queueReconcile(&nsa)
			return
		}
	}
}

// queueReconcile queues a CR for reconciliation
func (f *FileEvents) queueReconcile(cr *metalk8sv1alpha1.NodeSolutionArchive) {
	f.logger.Info().Msgf("Found matching Custom Resource: %s, queueing for reconcile", cr.Name)
	f.reconcileChan <- event.GenericEvent{
		Object: cr,
	}
}
