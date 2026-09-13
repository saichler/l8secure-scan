// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sec

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

func CreateCertBundle() (string, string, string) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2025),
		Subject: pkix.Name{
			CommonName:    "saichler",
			Organization:  []string{"Layer8"},
			Country:       []string{"USA"},
			Province:      []string{"Santa Clara"},
			Locality:      []string{"San Jose"},
			StreetAddress: []string{"1993 Curtner Ave"},
			PostalCode:    []string{"95124"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		EmailAddresses:        []string{"saichler@gmail.com"},
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	caData, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		panic(err)
	}

	caPEM := &bytes.Buffer{}
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caData,
	})

	crt := &x509.Certificate{
		SerialNumber: big.NewInt(10005),
		Subject: pkix.Name{
			CommonName:    "www.layer8vibe.dev",
			Organization:  []string{"Layer8"},
			Country:       []string{"USA"},
			Province:      []string{"Santa Clara"},
			Locality:      []string{"San Jose"},
			StreetAddress: []string{"1993 Curtner Ave"},
			PostalCode:    []string{"95124"},
		},
		EmailAddresses: []string{"saichler@gmail.com"},
		IPAddresses:    localIPs(),
		NotBefore:      time.Now(),
		NotAfter:       time.Now().AddDate(10, 0, 0),
		SubjectKeyId:   []byte("Layer8Secret"),
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:       x509.KeyUsageDigitalSignature,
	}

	crtKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	crtData, err := x509.CreateCertificate(rand.Reader, crt, ca, &crtKey.PublicKey, caKey)
	if err != nil {
		panic(err)
	}

	crtPEM := &bytes.Buffer{}
	pem.Encode(crtPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crtData,
	})

	crtKeyPEM := &bytes.Buffer{}
	pem.Encode(crtKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(crtKey),
	})

	domainCert := base64.StdEncoding.EncodeToString(crtPEM.Bytes())
	privateKey := base64.StdEncoding.EncodeToString(crtKeyPEM.Bytes())
	publicKey := base64.StdEncoding.EncodeToString(caPEM.Bytes())

	return domainCert, privateKey, publicKey
}

func localIPs() []net.IP {
	ips := []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("127.0.0.1")}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
			ips = append(ips, ipNet.IP)
		}
	}
	return ips
}
