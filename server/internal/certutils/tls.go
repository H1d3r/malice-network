package certutils

import (
	"crypto/tls"
	"net"

	"github.com/chainreactors/malice-network/helper/implanttypes"
	ucert "github.com/chainreactors/utils/cert"
)

func WrapWithTls(lsn net.Listener, cert *implanttypes.CertConfig) (net.Listener, error) {
	pair, err := tls.X509KeyPair([]byte(cert.Cert), []byte(cert.Key))
	if err != nil {
		return nil, err
	}
	return tls.NewListener(lsn, TlsConfig(pair)), nil
}

func GetTlsConfig(config *implanttypes.CertConfig) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(config.Cert), []byte(config.Key))
	if err != nil {
		return nil, err
	}
	return TlsConfig(cert), nil
}

func GetMTlsConfig(serverCert *implanttypes.CertConfig, caCert *implanttypes.CertConfig) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(serverCert.Cert), []byte(serverCert.Key))
	if err != nil {
		return nil, err
	}
	pool, err := ucert.LoadCertPool([]byte(caCert.Cert))
	if err != nil {
		return nil, err
	}
	return ucert.NewTLSConfig(cert, ucert.TLSMutualAuth(pool)), nil
}

func TlsConfig(cert tls.Certificate) *tls.Config {
	return ucert.NewTLSConfig(cert)
}
