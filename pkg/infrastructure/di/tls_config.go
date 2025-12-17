package di

import (
	"crypto/tls"
)

func (c *Container) getExternTLSConfig() *tls.Config {
	if c.ExternTLSConfig == nil {
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
		}
	}
	return c.ExternTLSConfig
}

func (c *Container) getInternTLSConfig() *tls.Config {
	if c.InternTLSConfig == nil {
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
		}
	}
	return c.InternTLSConfig
}
