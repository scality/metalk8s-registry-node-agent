// nolint:dupl // normal to have the internal and external servers very similar
package di

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"os"
)

func (c *Container) getExternTLSConfig() *tls.Config {
	if c.ExternTLSConfig == nil {
		externClientCACertPEM, err := os.ReadFile(c.config.Extern.ServerAuthN.CACertFilePath)
		if err != nil {
			c.GetLogger().ErrorContext(
				c.ctx,
				"failed to read external client CA certificates",
				slog.String("extern_server_authn_ca_cert_file_path", c.config.Extern.ServerAuthN.CACertFilePath),
				slog.Any("error", err),
			)
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}

		externClientCAPool := x509.NewCertPool()
		if !externClientCAPool.AppendCertsFromPEM(externClientCACertPEM) {
			c.GetLogger().ErrorContext(c.ctx, "failed to append external client CA certificates to pool")
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
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
			c.GetLogger().ErrorContext(
				c.ctx,
				"failed to read internal client CA certificates",
				slog.String("intern_server_authn_ca_cert_file_path", c.config.Intern.ServerAuthN.CACertFilePath),
				slog.Any("error", err),
			)
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}

		internClientCAPool := x509.NewCertPool()
		if !internClientCAPool.AppendCertsFromPEM(internClientCACertPEM) {
			c.GetLogger().ErrorContext(c.ctx, "failed to append internal client CA certificates to pool")
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
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
			c.GetLogger().ErrorContext(
				c.ctx,
				"failed to read internal server CA certificate",
				slog.String("intern_client_tls_ca_cert_file_path", c.config.Intern.ClientTLS.CACertFilePath),
				slog.Any("error", err),
			)
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
		}

		internServerCAPool := x509.NewCertPool()
		if !internServerCAPool.AppendCertsFromPEM(internCACertPEM) {
			c.GetLogger().ErrorContext(c.ctx, "failed to append internal server CA certificates to pool")
			os.Exit(1) //nolint:revive // Fatal-equivalent for DI initialization failure.
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
