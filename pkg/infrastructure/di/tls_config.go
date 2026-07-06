// nolint:dupl // normal to have the internal and external servers very similar
package di

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func (c *Container) getExternTLSConfig() *tls.Config {
	if c.ExternTLSConfig == nil {
		externClientCACertPEM, err := os.ReadFile(c.config.Extern.ServerAuthN.CACertFilePath)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("extern_serverAuthN_ca_cert_file_path", c.config.Extern.ServerAuthN.CACertFilePath).
				Msg("failed to read external client CA certificates")
		}

		externClientCAPool := x509.NewCertPool()
		if !externClientCAPool.AppendCertsFromPEM(externClientCACertPEM) {
			c.GetLogger().Fatal().Err(err).
				Msg("failed to append external client CA certificates to pool")
		}

		c.ExternTLSConfig = &tls.Config{
			GetCertificate: c.getExternServerCertWatcher().GetCertificate,
			ClientAuth:     tls.RequireAndVerifyClientCert,
			ClientCAs:      externClientCAPool,
		}
	}
	return c.ExternTLSConfig
}

func (c *Container) getInternTLSConfig() *tls.Config {
	if c.InternTLSConfig == nil {
		internClientCACertPEM, err := os.ReadFile(c.config.Intern.ServerAuthN.CACertFilePath)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("intern_serverAuthN_ca_cert_file_path", c.config.Intern.ServerAuthN.CACertFilePath).
				Msg("failed to read internal client CA certificates")
		}

		internClientCAPool := x509.NewCertPool()
		if !internClientCAPool.AppendCertsFromPEM(internClientCACertPEM) {
			c.GetLogger().Fatal().Err(err).
				Msg("failed to append internal client CA certificates to pool")
		}

		c.InternTLSConfig = &tls.Config{
			GetCertificate: c.getInternServerCertWatcher().GetCertificate,
			ClientAuth:     tls.RequireAndVerifyClientCert,
			ClientCAs:      internClientCAPool,
		}
	}
	return c.InternTLSConfig
}

func (c *Container) getInternTLSClientConfig() *tls.Config {
	if c.InternTLSClientConfig == nil {
		internClientCertWatcher := c.getInternClientCertWatcher()

		internCACertPEM, err := os.ReadFile(c.config.Intern.ClientTLS.CACertFilePath)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("intern_clientTLS_ca_cert_file_path", c.config.Intern.ClientTLS.CACertFilePath).
				Msg("failed to read internal server CA certificate")
		}

		internServerCAPool := x509.NewCertPool()
		if !internServerCAPool.AppendCertsFromPEM(internCACertPEM) {
			c.GetLogger().Fatal().Err(err).
				Msg("failed to append internal server CA certificates to pool")
		}

		c.InternTLSClientConfig = &tls.Config{
			GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return internClientCertWatcher.GetCertificate(nil)
			},
			RootCAs: internServerCAPool,
		}
	}
	return c.InternTLSClientConfig
}
