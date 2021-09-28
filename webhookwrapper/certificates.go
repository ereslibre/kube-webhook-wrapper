package webhookwrapper

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

type certificate struct {
	Certificate string
	PrivateKey  string
	certificate *x509.Certificate
	privateKey  *rsa.PrivateKey
}

func newCertificateAuthority(authorityName string) (*certificate, error) {
	privateKey, err := newPrivateKey(1024)
	if err != nil {
		return nil, err
	}
	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return nil, err
	}
	caCertificate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:    authorityName,
			Organization:  []string{""},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caCertificateBytes, err := x509.CreateCertificate(rand.Reader, &caCertificate, &caCertificate, &privateKey.Key.PublicKey, privateKey.Key)
	if err != nil {
		return nil, err
	}
	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertificateBytes,
	})
	if err != nil {
		return nil, err
	}
	return &certificate{
		Certificate: caPEM.String(),
		PrivateKey:  privateKey.PrivateKey,
		certificate: &caCertificate,
		privateKey:  privateKey.Key,
	}, nil
}

func (certificate *certificate) createCertificate(commonName string, organization []string, extraSANs []string) (string, string, error) {
	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return "", "", err
	}
	sansHosts := []string{"localhost"}
	sansIps := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	knownSans := map[string]struct{}{
		"localhost":               {},
		"127.0.0.1":               {},
		net.IPv6loopback.String(): {},
	}
	for _, extraSAN := range extraSANs {
		if _, exists := knownSans[extraSAN]; exists {
			continue
		}
		if ip := net.ParseIP(extraSAN); ip != nil {
			sansIps = append(sansIps, ip)
		} else {
			sansHosts = append(sansHosts, extraSAN)
		}
		knownSans[extraSAN] = struct{}{}
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", "", err
	}
	newCertificate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:    commonName,
			Organization:  organization,
			Country:       []string{"Some Country"},
			Province:      []string{"Some Province"},
			Locality:      []string{"Some Locality"},
			StreetAddress: []string{"Some StreetAddress"},
			PostalCode:    []string{"Some PostalCode"},
		},
		DNSNames:     sansHosts,
		IPAddresses:  sansIps,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certificateBytes, err := x509.CreateCertificate(rand.Reader, &newCertificate, certificate.certificate, &privateKey.PublicKey, certificate.privateKey)
	if err != nil {
		return "", "", err
	}
	certificatePEM := new(bytes.Buffer)
	err = pem.Encode(certificatePEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificateBytes,
	})
	if err != nil {
		return "", "", err
	}
	certificatePrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certificatePrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return "", "", err
	}
	return certificatePEM.String(), certificatePrivKeyPEM.String(), nil
}
