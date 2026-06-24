package certs

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"

	ucert "github.com/chainreactors/utils/cert"
	"github.com/chainreactors/malice-network/helper/codenames"
)

func RandomSubject(commonName string) *pkix.Name {
	codenames.SetupCodenames()
	return ucert.RandomSubjectWith(commonName,
		ucert.WithWordLists(codenames.Adjectives, codenames.Nouns),
	)
}

func ExtractCertificateSubject(certPEM string) (*pkix.Name, error) {
	if certPEM == "" {
		return nil, nil
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	subject := &pkix.Name{
		CommonName: cert.Subject.CommonName,
	}
	if len(cert.Subject.Organization) > 0 {
		subject.Organization = cert.Subject.Organization
	} else {
		subject.Organization = []string{""}
	}
	if len(cert.Subject.Country) > 0 {
		subject.Country = cert.Subject.Country
	} else {
		subject.Country = []string{""}
	}
	if len(cert.Subject.Locality) > 0 {
		subject.Locality = cert.Subject.Locality
	} else {
		subject.Locality = []string{""}
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		subject.OrganizationalUnit = cert.Subject.OrganizationalUnit
	} else {
		subject.OrganizationalUnit = []string{""}
	}
	if len(cert.Subject.StreetAddress) > 0 {
		subject.StreetAddress = cert.Subject.StreetAddress
	} else {
		subject.StreetAddress = []string{""}
	}
	if len(cert.Subject.Province) > 0 {
		subject.Province = cert.Subject.Province
	}

	return subject, nil
}

func FormatSubject(name, certType, certPEM string) (string, error) {
	subject, err := ExtractCertificateSubject(certPEM)
	if err != nil {
		return "", nil
	}
	ouStr := ""
	if len(subject.OrganizationalUnit) > 0 {
		ouStr = subject.OrganizationalUnit[0]
	}
	stStr := ""
	if len(subject.Province) > 0 {
		stStr = subject.Province[0]
	}
	return fmt.Sprintf("cert %s (type: %s) generate success, CN: %s, O: %s, C: %s, L: %s, OU: %s, ST: %s",
		name, certType, subject.CommonName, subject.Organization[0], subject.Country[0], subject.Locality[0],
		ouStr, stStr), nil
}
