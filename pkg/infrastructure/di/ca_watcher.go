package di

import (
	"log/slog"
	"os"

	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/capool"
)

// newCAWatcher returns a CA pool watcher for path, or exits when the file is not valid.
func (c *Container) newCAWatcher(path string, name string) *capool.Watcher {
	watcher, err := capool.New(path, c.GetLogger())
	if err != nil {
		c.GetLogger().ErrorContext(
			c.ctx,
			"failed to initialize CA pool watcher",
			slog.String("name", name),
			slog.String("path", path),
			slog.Any("error", err),
		)
		os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
	}
	return watcher
}

// getExternClientCAWatcher returns the watcher of the CA that checks
// client certificates on the external server.
func (c *Container) getExternClientCAWatcher() *capool.Watcher {
	if c.externClientCAWatcher == nil {
		c.externClientCAWatcher = c.newCAWatcher(
			c.config.Extern.ServerAuthN.CACertFilePath,
			"external client CA",
		)
	}
	return c.externClientCAWatcher
}

// getInternClientCAWatcher returns the watcher of the CA that checks
// client certificates on the internal server.
func (c *Container) getInternClientCAWatcher() *capool.Watcher {
	if c.internClientCAWatcher == nil {
		c.internClientCAWatcher = c.newCAWatcher(
			c.config.Intern.ServerAuthN.CACertFilePath,
			"internal client CA",
		)
	}
	return c.internClientCAWatcher
}

// getInternServerCAWatcher returns the watcher of the CA that checks
// the server certificate of peer nodes.
func (c *Container) getInternServerCAWatcher() *capool.Watcher {
	if c.internServerCAWatcher == nil {
		c.internServerCAWatcher = c.newCAWatcher(
			c.config.Intern.ClientTLS.CACertFilePath,
			"internal server CA",
		)
	}
	return c.internServerCAWatcher
}

// GetCAWatchers returns the three CA pool watchers (external client,
// internal client, internal server). Each one must be started to
// follow CA renewals.
func (c *Container) GetCAWatchers() []*capool.Watcher {
	return []*capool.Watcher{
		c.getExternClientCAWatcher(),
		c.getInternClientCAWatcher(),
		c.getInternServerCAWatcher(),
	}
}
