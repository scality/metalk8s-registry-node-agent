package di

import (
	"crypto/tls"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

func (c *Container) getExternTLSConfig() (*tls.Config, error) {
	if c.ExternTLSConfig == nil {
		externServerCert, err := tls.LoadX509KeyPair(
			c.config.Extern.ServerTLS.CertFilePath,
			c.config.Extern.ServerTLS.KeyFilePath,
		)
		if err != nil {
			return nil, errors.From(domain.ErrInternal).
				WithIdentifier(500000).
				WithDetail("failed to read external server certificate file").
				WithProperty("extern_serverTLS_cert_file_path", c.config.Extern.ServerTLS.CertFilePath).
				WithProperty("extern_serverTLS_key_file_path", c.config.Extern.ServerTLS.KeyFilePath).
				CausedBy(err).
				Throw()
		}

		c.ExternTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{externServerCert},
		}
	}
	return c.ExternTLSConfig, nil
}

func (c *Container) getInternTLSConfig() (*tls.Config, error) {
	if c.InternTLSConfig == nil {
		internServerCert, err := tls.LoadX509KeyPair(
			c.config.Intern.ServerTLS.CertFilePath,
			c.config.Intern.ServerTLS.KeyFilePath,
		)
		if err != nil {
			return nil, errors.From(domain.ErrInternal).
				WithIdentifier(500000).
				WithDetail("failed to read internal server certificate file").
				WithProperty("intern_serverTLS_cert_file_path", c.config.Intern.ServerTLS.CertFilePath).
				WithProperty("intern_serverTLS_key_file_path", c.config.Intern.ServerTLS.KeyFilePath).
				CausedBy(err).
				Throw()
		}

		c.InternTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{internServerCert},
		}
	}
	return c.InternTLSConfig, nil
}
