//nolint:dupl // Duplication is fine in DI packages.
package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/usecase"

func (c *Container) getUploadPartUseCase() *usecase.UploadPart {
	if c.uploadPartUseCase == nil {
		c.uploadPartUseCase = usecase.NewUploadPart(
			c.GetLogger(),
			c.getStoragePartUploader(),
		)
	}

	return c.uploadPartUseCase
}

func (c *Container) GetInitializeSessionUseCase() *usecase.InitializeSession {
	if c.initializeSessionUseCase == nil {
		c.initializeSessionUseCase = usecase.NewInitializeSession(
			c.GetLogger(),
			c.getStorageSessionInitializer(),
		)
	}

	return c.initializeSessionUseCase
}

func (c *Container) GetRemoveArtifactUseCase() *usecase.RemoveArtifact {
	if c.removeArtifactUseCase == nil {
		c.removeArtifactUseCase = usecase.NewRemoveArtifact(
			c.GetLogger(),
			c.getStorageArtifactRemover(),
		)
	}
	return c.removeArtifactUseCase
}

func (c *Container) GetValidateArtifactUseCase() *usecase.ValidateArtifact {
	if c.validateArtifactUseCase == nil {
		c.validateArtifactUseCase = usecase.NewValidateArtifact(
			c.GetLogger(),
			c.getStorageArtifactValidator(),
		)
	}
	return c.validateArtifactUseCase
}

func (c *Container) GetRemoveSessionUseCase() *usecase.RemoveSession {
	if c.removeSessionUseCase == nil {
		c.removeSessionUseCase = usecase.NewRemoveSession(
			c.GetLogger(),
			c.getStorageSessionRemover(),
		)
	}
	return c.removeSessionUseCase
}
