package di

import (
	"crypto/tls"
)

// getExternTLSConfig returns the TLS config of the external server. The client
// certificate is required and checked against the current external client CA.
func (c *Container) getExternTLSConfig() *tls.Config {
	if c.ExternTLSConfig == nil {
		c.ExternTLSConfig = &tls.Config{
			GetCertificate: c.getExternServerCertWatcher().GetCertificate,
			// The chain is checked by VerifyConnection, with a CA pool that follows renewals.
			ClientAuth:       tls.RequireAnyClientCert,
			VerifyConnection: c.getExternClientCAWatcher().VerifyClientConnection,
		}
	}
	return c.ExternTLSConfig
}

// getInternTLSConfig returns the TLS config of the internal server. The client
// certificate is required and checked against the current internal client CA.
func (c *Container) getInternTLSConfig() *tls.Config {
	if c.InternTLSConfig == nil {
		c.InternTLSConfig = &tls.Config{
			GetCertificate: c.getInternServerCertWatcher().GetCertificate,
			// The chain is checked by VerifyConnection, with a CA pool that follows renewals.
			ClientAuth:       tls.RequireAnyClientCert,
			VerifyConnection: c.getInternClientCAWatcher().VerifyClientConnection,
		}
	}
	return c.InternTLSConfig
}

// getInternTLSClientConfig returns the TLS config used to download from peer
// nodes. The server certificate is checked against the current internal server CA.
func (c *Container) getInternTLSClientConfig() *tls.Config {
	if c.InternTLSClientConfig == nil {
		internClientCertWatcher := c.getInternClientCertWatcher()

		c.InternTLSClientConfig = &tls.Config{
			GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return internClientCertWatcher.GetCertificate(nil)
			},
			// The default check would use a fixed RootCAs pool. VerifyConnection
			// checks the chain and the host name with a pool that follows renewals.
			InsecureSkipVerify: true, //nolint:gosec // Replaced by VerifyConnection below.
			VerifyConnection:   c.getInternServerCAWatcher().VerifyServerConnection,
		}
	}
	return c.InternTLSClientConfig
}
