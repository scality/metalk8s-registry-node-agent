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

		externServerCert, err := tls.LoadX509KeyPair(
			c.config.Extern.ServerTLS.CertFilePath,
			c.config.Extern.ServerTLS.KeyFilePath,
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("extern_serverTLS_cert_file_path", c.config.Extern.ServerTLS.CertFilePath).
				Str("extern_serverTLS_key_file_path", c.config.Extern.ServerTLS.KeyFilePath).
				Msg("failed to read external server certificate file")
		}

		c.ExternTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{externServerCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    externClientCAPool,
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

		internServerCert, err := tls.LoadX509KeyPair(
			c.config.Intern.ServerTLS.CertFilePath,
			c.config.Intern.ServerTLS.KeyFilePath,
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("intern_serverTLS_cert_file_path", c.config.Intern.ServerTLS.CertFilePath).
				Str("intern_serverTLS_key_file_path", c.config.Intern.ServerTLS.KeyFilePath).
				Msg("failed to read internal server certificate file")
		}

		c.InternTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{internServerCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    internClientCAPool,
		}
	}
	return c.InternTLSConfig
}

func (c *Container) getInternTLSClientConfig() *tls.Config {
	if c.InternTLSClientConfig == nil {
		internClientCert, err := tls.LoadX509KeyPair(
			c.config.Intern.ClientAuthN.CertFilePath,
			c.config.Intern.ClientAuthN.KeyFilePath,
		)
		if err != nil {
			c.GetLogger().Fatal().Err(err).
				Str("intern_clientAuthN_cert_file_path", c.config.Intern.ClientAuthN.CertFilePath).
				Str("intern_clientAuthN_key_file_path", c.config.Intern.ClientAuthN.KeyFilePath).
				Msg("failed to read internal client certificate files")
		}

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
			Certificates: []tls.Certificate{internClientCert},
			RootCAs:      internServerCAPool,
		}
	}
	return c.InternTLSClientConfig
}
