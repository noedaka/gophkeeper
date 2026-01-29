package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"
)

// generateKey генерирует P256 приватный ключ
func generateKey() *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	return key
}

// savePEM
func savePEM(fileName string, blockType string, bytes []byte) {
	pemBlock := &pem.Block{Type: blockType, Bytes: bytes}
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	pem.Encode(file, pemBlock)
}

func main() {
	caPriv := generateKey()

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"My CA"},
			CommonName:   "My Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPriv.PublicKey, caPriv)
	if err != nil {
		log.Fatal(err)
	}

	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		log.Fatal(err)
	}

	savePEM("ca.crt", "CERTIFICATE", caDER)
	caPrivBytes, _ := x509.MarshalECPrivateKey(caPriv)
	savePEM("ca.key", "EC PRIVATE KEY", caPrivBytes)

	serverPriv := generateKey()

	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Gophkeeper"},
			CommonName:   "localhost",
		},
		DNSNames:  []string{"localhost"},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, &serverTemplate, caCert, &serverPriv.PublicKey, caPriv)
	if err != nil {
		log.Fatal(err)
	}

	savePEM("server.crt", "CERTIFICATE", serverDER)
	serverPrivBytes, _ := x509.MarshalECPrivateKey(serverPriv)
	savePEM("server.key", "EC PRIVATE KEY", serverPrivBytes)
}
