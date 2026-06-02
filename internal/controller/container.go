package controller

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

type containerInterface interface {
	GetRemoveSessionUseCase() *usecase.RemoveSession
	GetRemoveSolutionArchiveUseCase() *usecase.RemoveSolutionArchive
	GetValidateSolutionArchiveUseCase() *usecase.ValidateSolutionArchive
	GetInitializeSessionUseCase() *usecase.InitializeSession
	GetDownloadPartUseCase() *usecase.DownloadPart
	GetUnmountSolutionArchiveUseCase() *usecase.UnmountSolutionArchive
	GetMountSolutionArchiveUseCase() *usecase.MountSolutionArchive
}
