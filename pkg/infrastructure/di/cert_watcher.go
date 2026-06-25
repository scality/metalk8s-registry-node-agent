package di

import (
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
)

// getExternServerCertWatcher returns the certwatcher tracking the external
// server TLS key pair. The watcher reloads the certificate from disk whenever
// cert-manager rotates it, so live connections keep using a valid certificate.
func (c *Container) getExternServerCertWatcher() *certwatcher.CertWatcher {
	if c.externServerCertWatcher == nil {
		watcher, err := certwatcher.New(
			c.config.Extern.ServerTLS.CertFilePath,
			c.config.Extern.ServerTLS.KeyFilePath,
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("extern_serverTLS_cert_file_path", c.config.Extern.ServerTLS.CertFilePath).
				Str("extern_serverTLS_key_file_path", c.config.Extern.ServerTLS.KeyFilePath).
				Msg("failed to initialize external server certificate watcher")
		}

		c.externServerCertWatcher = watcher
	}
	return c.externServerCertWatcher
}

// getInternServerCertWatcher returns the certwatcher tracking the internal
// server TLS key pair.
func (c *Container) getInternServerCertWatcher() *certwatcher.CertWatcher {
	if c.internServerCertWatcher == nil {
		watcher, err := certwatcher.New(
			c.config.Intern.ServerTLS.CertFilePath,
			c.config.Intern.ServerTLS.KeyFilePath,
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("intern_serverTLS_cert_file_path", c.config.Intern.ServerTLS.CertFilePath).
				Str("intern_serverTLS_key_file_path", c.config.Intern.ServerTLS.KeyFilePath).
				Msg("failed to initialize internal server certificate watcher")
		}

		c.internServerCertWatcher = watcher
	}
	return c.internServerCertWatcher
}

// getInternClientCertWatcher returns the certwatcher tracking the internal
// client mTLS key pair used to pull archives from peer nodes.
func (c *Container) getInternClientCertWatcher() *certwatcher.CertWatcher {
	if c.internClientCertWatcher == nil {
		watcher, err := certwatcher.New(
			c.config.Intern.ClientAuthN.CertFilePath,
			c.config.Intern.ClientAuthN.KeyFilePath,
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("intern_clientAuthN_cert_file_path", c.config.Intern.ClientAuthN.CertFilePath).
				Str("intern_clientAuthN_key_file_path", c.config.Intern.ClientAuthN.KeyFilePath).
				Msg("failed to initialize internal client certificate watcher")
		}

		c.internClientCertWatcher = watcher
	}
	return c.internClientCertWatcher
}

// GetCertWatchers eagerly constructs and returns the three TLS key-pair
// watchers (external server, internal server, internal client) so the caller
// can register them as controller-runtime manager Runnables. Each watcher must
// be started (Start) for the renewal detection to be active.
func (c *Container) GetCertWatchers() []*certwatcher.CertWatcher {
	return []*certwatcher.CertWatcher{
		c.getExternServerCertWatcher(),
		c.getInternServerCertWatcher(),
		c.getInternClientCertWatcher(),
	}
}
