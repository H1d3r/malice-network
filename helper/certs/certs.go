package certs

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"

	ucert "github.com/chainreactors/utils/cert"
)

const (
	OperatorCA = iota + 1
	ListenerCA
	ImplantCA
	RootCA
)

const (
	RSAKey            = "rsa"
	RootName          = "Root"
	RootCert          = "root_ca.pem"
	RootKey           = "root_key.pem"
	ServerCert        = "server_crt.pem"
	ServerKey         = "server_key.pem"
	ForwardClientCert = "forward_client_crt.pem"
	ForwardClientKey  = "forward_client_key.pem"

	RootNamespace     = "root"
	ListenerNamespace = "listener"
	ClientNamespace   = "client"
)

const (
	Acme       = "acme"
	SelfSigned = "self_signed"
	Imported   = "imported"
)

var CertTypes = []string{
	Acme, SelfSigned, Imported,
}

func SaveToPEMFile(filename string, pemData []byte) error {
	return ucert.SaveToPEMFile(filename, pemData)
}

func RsaKeySize() int {
	return ucert.RandomKeySize()
}

func GenerateCACert(commonName string, subject *pkix.Name) ([]byte, []byte, error) {
	var opts []ucert.TemplateOption
	if subject != nil {
		opts = append(opts, ucert.WithFullSubject(*subject))
	} else {
		opts = append(opts, ucert.WithRandomSubject(commonName))
	}
	opts = append(opts, ucert.WithExtKeyUsages(x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth))
	return ucert.GenerateCACert(0, opts...)
}

func GenerateChildCert(commonName string, isClient bool, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	usage := x509.ExtKeyUsageServerAuth
	if isClient {
		usage = x509.ExtKeyUsageClientAuth
	}
	return GenerateChildCertWithUsages(commonName, []x509.ExtKeyUsage{usage}, caCert, caKey)
}

func GenerateChildCertWithUsages(commonName string, extKeyUsages []x509.ExtKeyUsage, caCert *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	opts := []ucert.TemplateOption{
		ucert.WithRandomSubject(commonName),
		ucert.WithExtKeyUsages(extKeyUsages...),
	}
	for _, u := range extKeyUsages {
		if u == x509.ExtKeyUsageServerAuth {
			opts = append(opts, ucert.WithAutoSAN(commonName))
			break
		}
	}
	return ucert.GenerateChildCert(0, caCert, caKey, opts...)
}
