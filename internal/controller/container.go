package controller

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

type containerInterface interface {
	GetRemoveSessionUseCase() *usecase.RemoveSession
	GetRemoveArtifactUseCase() *usecase.RemoveArtifact
	GetValidateArtifactUseCase() *usecase.ValidateArtifact
	GetInitializeSessionUseCase() *usecase.InitializeSession
}
