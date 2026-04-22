/*
Copyright 2025 Scality.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint:revive,staticcheck
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

const (
	prometheusOperatorVersion = "v0.77.1"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"

	certmanagerVersion = "v1.16.3"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) (string, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
}

// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsPrometheusCRDsInstalled checks if any Prometheus CRDs are installed
// by verifying the existence of key CRDs related to Prometheus.
func IsPrometheusCRDsInstalled() bool {
	// List of common Prometheus CRDs
	prometheusCRDs := []string{
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"prometheusagents.monitoring.coreos.com",
	}

	cmd := exec.Command("kubectl", "get", "crds", "-o", "custom-columns=NAME:.metadata.name")
	output, err := Run(cmd)
	if err != nil {
		return false
	}
	crdList := GetNonEmptyLines(output)
	for _, crd := range prometheusCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := exec.Command("kubectl", "apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = exec.Command("kubectl", "wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// IsCertManagerCRDsInstalled checks if any Cert Manager CRDs are installed
// by verifying the existence of key CRDs related to Cert Manager.
func IsCertManagerCRDsInstalled() bool {
	// List of common Cert Manager CRDs
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	// Execute the kubectl command to get all CRDs
	cmd := exec.Command("kubectl", "get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	// Check if any of the Cert Manager CRDs are present
	crdList := GetNonEmptyLines(output)
	for _, crd := range certManagerCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// LoadImageToKindClusterWithName loads a local docker image to the kind cluster
func LoadImageToKindClusterWithName(name string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", name, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
func UncommentCode(filename, target, prefix string) error {
	// false positive
	// nolint:gosec
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	strContent := string(content)

	idx := strings.Index(strContent, target)
	if idx < 0 {
		return fmt.Errorf("unable to find the code %q to be uncomment", target)
	}

	out := new(bytes.Buffer)
	_, err = out.Write(content[:idx])
	if err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(target))
	if !scanner.Scan() {
		return nil
	}
	for {
		if _, err = out.WriteString(strings.TrimPrefix(scanner.Text(), prefix)); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
		// Avoid writing a newline in case the previous line was the last in target.
		if !scanner.Scan() {
			break
		}
		if _, err = out.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if _, err = out.Write(content[idx+len(target):]); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	// false positive
	// nolint:gosec
	if err = os.WriteFile(filename, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	return nil
}

func GetHTTPExternClient(TLSConfig *tls.Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: TLSConfig,
		},
	}
}

func GetGeneratedHTTPExternClient(
	addr, apiPath string, httpClient *http.Client,
) (*extern.ClientWithResponses, error) {
	return extern.NewClientWithResponses(
		"https://localhost"+addr+apiPath,
		extern.WithHTTPClient(httpClient),
	)
}

// GenerateFakeTLSConfig creates a *tls.Config with a new,
// self-signed certificate and private key generated in memory.
// This returns a server TLS config suitable for mTLS.
func GenerateFakeTLSConfig() (*tls.Config, error) {
	// 1. Generate a new private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// 2. Create a certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1), // Simple serial number
		Subject: pkix.Name{
			Organization: []string{"My Fake Corp"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365), // Valid for 1 year

		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		// Add IP addresses and DNS names the certificate should be valid for.
		// "localhost" is common for testing.
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},
	}

	// 3. Create the self-signed certificate
	// We use the same template as the parent and child, and the same private key to sign it.
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, err
	}

	// 4. Create a tls.Certificate struct
	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER}, // The DER-encoded certificate
		PrivateKey:  privKey,           // The private key
	}

	// 5. Create a certificate pool and add the certificate as a trusted CA
	rawCert := tlsCert.Certificate[0]
	cert, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(cert)

	// 6. Return the tls.Config for server with mTLS
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}, nil
}

// GenerateFakeTLSClientConfig creates a *tls.Config for the client side
// that matches the server config returned by GenerateFakeTLSConfig.
// It uses the same certificate for client authentication and trusts the same CA.
func GenerateFakeTLSClientConfig(serverConfig *tls.Config) *tls.Config {
	// Use the same certificate from the server config for client authentication
	// and set RootCAs to trust the server's certificate
	return &tls.Config{
		Certificates: serverConfig.Certificates,
		RootCAs:      serverConfig.ClientCAs, // Trust the same CA that the server uses
	}
}

func randomBytes(length int64) []byte {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	_, _ = rand.Read(b) // nolint:errcheck // No need to check the error.
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return b
}

func RandomString(length int64) string {
	return string(randomBytes(length))
}

func RandomInt(min, max int) int {
	n := max - min + 1
	b, _ := rand.Int(rand.Reader, big.NewInt(int64(n))) // nolint:errcheck // No need to check the error.
	return int(b.Int64()) + min
}

// MakeISOFile creates a proper ISO file for testing
func MakeISOFile(isoFilePath string, size int64) (string, error) {
	// Create a directory with test content
	isoContentPath, err := os.MkdirTemp("/tmp", "iso-file-*")
	if err != nil {
		return "", fmt.Errorf("failed to create test content directory: %w", err)
	}
	defer os.RemoveAll(isoContentPath) // nolint: errcheck // No error check on defer.

	// Create a test file in the directory
	testFile := filepath.Join(isoContentPath, "test.txt")
	err = os.WriteFile(testFile, randomBytes(size), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create test file: %w", err)
	}

	// Create ISO using genisoimage/mkisofs
	cmd := exec.Command("genisoimage", "-o", isoFilePath, "-V", "METALK8S", "-r", "-J", isoContentPath)
	_, err = Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to create ISO file: %w", err)
	}

	// Calculate the checksum of the ISO file
	isoData, err := os.ReadFile(isoFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read ISO file: %w", err)
	}
	hash := sha256.Sum256(isoData)
	checksum := hex.EncodeToString(hash[:])
	return checksum, nil
}

// MakeWrongISOFile creates a wrong ISO file for testing
func MakeWrongISOFile(isoFilePath string, size int64) (string, error) {
	err := os.WriteFile(isoFilePath, randomBytes(size), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create test file: %w", err)
	}

	// Calculate the checksum of the ISO file
	isoData, err := os.ReadFile(isoFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read ISO file: %w", err)
	}
	hash := sha256.Sum256(isoData)
	checksum := hex.EncodeToString(hash[:])
	return checksum, nil
}

func SaveFile(
	fullFileName string,
	content io.Reader,
	perm os.FileMode,
) error {
	return library.SaveFile(fullFileName, content, perm) // nolint:wrapcheck // No need to wrap the error.
}
