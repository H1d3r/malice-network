package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	certFile = "cert.pem"
	keyFile  = "key.pem"
	caFile   = "ca-cert.pem"
)

func DeleteCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeleteCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert %s delete success\n", certName)
	return nil
}

func InspectCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	cert, err := con.Rpc.DownloadCertificate(con.Context(), &clientpb.Cert{Name: certName})
	if err != nil {
		return err
	}
	printCert(cert)
	return nil
}

func VerifyCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	cert, err := con.Rpc.DownloadCertificate(con.Context(), &clientpb.Cert{Name: certName})
	if err != nil {
		return err
	}
	notBefore, notAfter, err := getCertExpireTime(cert.GetCert().GetCert())
	if err != nil {
		return err
	}
	now := time.Now()
	if now.Before(notBefore) {
		return fmt.Errorf("cert %s is not valid before %s", certName, notBefore.Format(time.RFC3339))
	}
	if now.After(notAfter) {
		return fmt.Errorf("cert %s expired at %s", certName, notAfter.Format(time.RFC3339))
	}
	if cert.GetCert().GetKey() != "" {
		if _, err := tls.X509KeyPair([]byte(cert.GetCert().GetCert()), []byte(cert.GetCert().GetKey())); err != nil {
			return fmt.Errorf("invalid certificate key pair: %w", err)
		}
	}
	con.Log.Infof("cert %s is valid until %s\n", certName, notAfter.Format("2006-01-02 15:04:05"))
	return nil
}

func RenewCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	domain, _ := cmd.Flags().GetString("domain")
	provider, _ := cmd.Flags().GetString("provider")
	email, _ := cmd.Flags().GetString("email")
	caURL, _ := cmd.Flags().GetString("ca-url")
	if domain == "" {
		cert, err := con.Rpc.DownloadCertificate(con.Context(), &clientpb.Cert{Name: certName})
		if err != nil {
			return err
		}
		domain = cert.GetDomain()
		if domain == "" {
			domain = cert.GetCert().GetName()
		}
	}
	if domain == "" {
		return fmt.Errorf("domain is required for ACME renew")
	}
	_, err := con.Rpc.ObtainAcmeCert(con.Context(), &clientpb.AcmeRequest{
		Domain:   domain,
		Provider: provider,
		Email:    email,
		CaUrl:    caURL,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert %s renew requested for %s\n", certName, domain)
	return nil
}

func ListRefsCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	refs := map[string]struct{}{}
	websites, err := con.Rpc.ListWebsites(con.Context(), &clientpb.Listener{})
	if err != nil {
		return err
	}
	for _, p := range websites.GetPipelines() {
		if pipelineUsesCert(p, certName) {
			refs[p.GetListenerId()+":"+p.GetName()] = struct{}{}
		}
	}
	pipelines, err := con.Rpc.ListPipelines(con.Context(), &clientpb.Listener{})
	if err != nil {
		return err
	}
	for _, p := range pipelines.GetPipelines() {
		if pipelineUsesCert(p, certName) {
			refs[p.GetListenerId()+":"+p.GetName()] = struct{}{}
		}
	}
	if len(refs) == 0 {
		con.Log.Infof("cert %s has no pipeline references\n", certName)
		return nil
	}
	keys := make([]string, 0, len(refs))
	for ref := range refs {
		keys = append(keys, strings.TrimPrefix(ref, ":"))
	}
	con.Log.Console(strings.Join(keys, "\n") + "\n")
	return nil
}

func PruneExpiredCmd(cmd *cobra.Command, con *core.Console) error {
	certs, err := con.Rpc.GetAllCertificates(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	cutoff := time.Now()
	pruned := 0
	for _, c := range certs.GetCerts() {
		name := c.GetCert().GetName()
		if name == "" {
			continue
		}
		_, notAfter, err := getCertExpireTime(c.GetCert().GetCert())
		if err != nil || notAfter.After(cutoff) {
			continue
		}
		if _, err := con.Rpc.DeleteCertificate(con.Context(), &clientpb.Cert{Name: name}); err != nil {
			return err
		}
		pruned++
	}
	con.Log.Infof("pruned %d expired certs\n", pruned)
	return nil
}

func pipelineUsesCert(p *clientpb.Pipeline, certName string) bool {
	if p == nil || certName == "" {
		return false
	}
	if p.GetCertName() == certName {
		return true
	}
	if p.GetTls() != nil && p.GetTls().GetCert().GetName() == certName {
		return true
	}
	return false
}

func UpdateCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	certPath, _ := cmd.Flags().GetString("cert")
	keyPath, _ := cmd.Flags().GetString("key")
	certType, _ := cmd.Flags().GetString("type")
	caPath, _ := cmd.Flags().GetString("ca-cert")
	comment, _ := cmd.Flags().GetString("comment")
	var cert, key, ca string
	var err error
	if certPath != "" || keyPath != "" {
		if certPath == "" || keyPath == "" {
			return fmt.Errorf("cert and key must be provided together")
		}
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			return err
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			return err
		}
	}
	if caPath != "" {
		ca, err = cryptography.ProcessPEM(caPath)
		if err != nil {
			return err
		}
	}
	_, err = con.Rpc.UpdateCertificate(con.Context(), &clientpb.TLS{
		Ca: &clientpb.Cert{
			Cert: ca,
		},
		Cert: &clientpb.Cert{
			Name:    certName,
			Cert:    cert,
			Key:     key,
			Type:    certType,
			Comment: comment,
		},
	})
	if err != nil {
		return err
	}
	con.Log.Infof("cert update %s success\n", certName)
	return nil
}

func DownloadCmd(cmd *cobra.Command, con *core.Console) error {
	certName := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	cert, err := con.Rpc.DownloadCertificate(con.Context(), &clientpb.Cert{
		Name: certName,
	})
	if err != nil {
		return err
	}
	printCert(cert)
	var path string
	if output != "" {
		path = filepath.Join(assets.GetTempDir(), output)
	} else {
		path = filepath.Join(assets.GetTempDir(), certName)
	}
	err = os.MkdirAll(path, 0700)
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(filepath.Join(path, certFile), []byte(cert.Cert.Cert))
	if err != nil {
		return err
	}
	err = certs.SaveToPEMFile(filepath.Join(path, keyFile), []byte(cert.Cert.Key))
	if err != nil {
		return err
	}
	if cert.Ca.Cert != "" {
		err = certs.SaveToPEMFile(filepath.Join(path, caFile), []byte(cert.Ca.Cert))
		if err != nil {
			return err
		}
	}
	con.Log.Infof("cert save in %s\n", path)
	return nil
}

func GetCertCmd(cmd *cobra.Command, con *core.Console) error {
	certs, err := con.Rpc.GetAllCertificates(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(certs.Certs) > 0 {
		printCerts(certs, con)
	} else {
		con.Log.Infof("no cert\n")
	}
	return nil
}

func printCert(cert *clientpb.TLS) {
	certBody := cert.GetCert()
	if certBody == nil {
		certBody = &clientpb.Cert{}
	}
	_, notAfter, err := getCertExpireTime(certBody.GetCert())
	expireStr := ""
	if err == nil {
		expireStr = notAfter.Format("2006-01-02 15:04:05")
	}
	subject := cert.GetCertSubject()
	if subject == nil {
		subject = &clientpb.CertificateSubject{}
	}
	certMap := map[string]interface{}{
		"Name":               certBody.Name,
		"Type":               certBody.Type,
		"Organization":       subject.O,
		"Country":            subject.C,
		"Locality":           subject.L,
		"OrganizationalUnit": subject.Ou,
		"StreetAddress":      subject.St,
		"Expire":             expireStr,
		"Comment":            certBody.Comment,
	}
	orderedKeys := []string{"Name", "Type", "Organization", "Country", "Locality", "OrganizationalUnit", "StreetAddress", "Expire", "Comment"}
	tui.RenderKVWithOptions(certMap, orderedKeys, tui.KVOptions{ShowHeader: true})
}

func printCerts(certs *clientpb.Certs, con *core.Console) {
	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Type", "Type", 11),
		table.NewFlexColumn("Organization", "Organization", 1),
		table.NewColumn("Country", "Country", 10),
		table.NewColumn("Locality", "Locality", 10),
		table.NewFlexColumn("OrganizationalUnit", "Organizational Unit", 1),
		table.NewFlexColumn("StreetAddress", "Street Address", 1),
		table.NewColumn("Expire", "Expire", 25),
		table.NewFlexColumn("Comment", "Comment", 1),
	}, true)

	for _, cert := range certs.Certs {
		certBody := cert.GetCert()
		if certBody == nil {
			certBody = &clientpb.Cert{}
		}
		_, notAfter, err := getCertExpireTime(certBody.GetCert())
		expireStr := ""
		if err == nil {
			expireStr = notAfter.Format("2006-01-02 15:04:05")
		}
		subject := cert.GetCertSubject()
		if subject == nil {
			subject = &clientpb.CertificateSubject{}
		}
		row := table.NewRow(table.RowData{
			"Name":               certBody.Name,
			"Type":               certBody.Type,
			"Organization":       subject.O,
			"Country":            subject.C,
			"Locality":           subject.L,
			"OrganizationalUnit": subject.Ou,
			"StreetAddress":      subject.St,
			"Expire":             expireStr,
			"Comment":            certBody.Comment,
		})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}

func getCertExpireTime(certPEM string) (notBefore, notAfter time.Time, err error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		err = errors.New("failed to parse certificate PEM")
		return
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return
	}
	return cert.NotBefore, cert.NotAfter, nil
}
