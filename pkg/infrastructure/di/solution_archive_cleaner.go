package di

import "github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/solutionarchivecleaner"

func (c *Container) GetStorageSolutionArchiveCleaner() *solutionarchivecleaner.SolutionArchiveCleaner {
	if c.solutionArchiveCleaner == nil {
		c.solutionArchiveCleaner = solutionarchivecleaner.NewSolutionArchiveCleaner(
			c.baseCtx,
			c.GetLogger(),
			c.GetFSSolutionArchiveStorage(),
			c.deleteChan,
		)
	}

	return c.solutionArchiveCleaner
}
