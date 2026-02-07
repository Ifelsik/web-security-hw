package proxy

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"os/exec"
	"sync"

	"github.com/google/uuid"
	"github.com/ifelsik/mitm-proxy/internal/utils/fileutil"
)

const (
	caCertPath        = "../certs/ifelser-mitm-ca.crt"
	caKeyPath         = "../certs/ifelser-mitm-ca.key"
	certGeneratorPath = "../scripts/gen_cert.sh"
	certPathDir       = "../certs"
	certPrivateKey    = certPathDir + "/" + "cert.key"
)

func generateCertTLS(domain string) (string, error) {
	id, _ := uuid.NewV7()
	serialNumber := "0x" + hex.EncodeToString(id[:])
	certPath := certPathDir + "/" + domain

	cmd := exec.Command(certGeneratorPath, caCertPath, caKeyPath, certPath, domain, serialNumber, certPrivateKey)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("generate certificate: %w", err)
	}
	return certPath + ".crt", nil
}

func loadCertificate(domain string) (tls.Certificate, error) {
	certPath := certPathDir + "/" + domain
	return tls.LoadX509KeyPair(certPath, certPrivateKey)
}

type CertCache struct {
	certs map[string]*tls.Certificate

	mu *sync.RWMutex
}

func NewCertCache() *CertCache {
	return &CertCache{
		certs: make(map[string]*tls.Certificate),
		mu:    &sync.RWMutex{},
	}
}

func (c *CertCache) GetOrCreate(sni string) (*tls.Certificate, error) {
	cert, ok := c.Get(sni)
	if ok {
		return cert, nil
	}

	certPath, err := generateCertTLS(sni)
	if err != nil {
		return nil, err
	}

	err = c.LoadFile(certPath, certPrivateKey)
	if err != nil {
		return nil, err
	}

	cert, _ = c.Get(sni)
	return cert, nil
}

func (c *CertCache) Get(sni string) (*tls.Certificate, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cert, ok := c.certs[sni]
	return cert, ok
}

func (c *CertCache) Put(domain string, cert *tls.Certificate) {
	if cert == nil {
		return
	}
	c.mu.Lock()
	c.certs[domain] = cert
	c.mu.Unlock()
}

func (c *CertCache) Load() error {
	files, err := fileutil.ListFiles(certPathDir)
	if err != nil {
		return fmt.Errorf("load certificates: %w", err)
	}

	for _, certFile := range files {
		certFile = certPathDir + "/" + certFile
		// That comparison looks shitty but caused by certs store.
		// Change requires refactor project structure or .sh scripts.
		if certFile == caCertPath ||
			certFile == caKeyPath ||
			certFile == certPrivateKey {
			continue
		}
		c.LoadFile(certFile, certPrivateKey)
	}
	return nil
}

func (c *CertCache) LoadFile(cert, key string) error {
	crt, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("load certificate: %w", err)
	}
	domain := fileutil.Filename(cert)
	c.Put(domain, &crt)
	return nil
}

func (c *CertCache) Array() []tls.Certificate {
	certs := make([]tls.Certificate, 0, len(c.certs))
	c.mu.RLock()
	for _, v := range c.certs {
		certs = append(certs, *v)
	}
	c.mu.RUnlock()
	return certs
}
