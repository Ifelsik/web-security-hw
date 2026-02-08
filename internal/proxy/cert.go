package proxy

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/ifelsik/mitm-proxy/internal/config"
	"github.com/ifelsik/mitm-proxy/internal/utils/fileutil"
)

type CertProvider struct {
	conf  config.TLS
	cache *CertCache
	cg    *CertGenerator
}

func NewCertProvider(conf config.TLS) *CertProvider {
	return &CertProvider{
		conf:  conf,
		cache: NewCertCache(),
		cg: &CertGenerator{
			conf: conf,
		},
	}
}

func (cp *CertProvider) LoadFile(certPath, keyPath string) error {
	crt, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("load certificate: %w", err)
	}
	domain := fileutil.Filename(certPath)
	cp.cache.Put(domain, &crt)
	return nil
}

func (cp *CertProvider) Load() error {
	files, err := fileutil.ListFiles(cp.conf.CertsDir)
	if err != nil {
		return fmt.Errorf("load certificates: %w", err)
	}

	for _, certFile := range files {
		certFile = filepath.Join(cp.conf.CertsDir, certFile)
		// That comparison looks shitty but caused by certs store.
		// Change requires refactor project structure or .sh scripts.
		if certFile == cp.conf.CACert ||
			certFile == cp.conf.CAKey ||
			certFile == cp.conf.CertKey {
			continue
		}
		_ = cp.LoadFile(certFile, cp.conf.CertKey)
	}
	return nil
}

func (cp *CertProvider) GenerateIfNotExist(sni string) error {
	_, ok := cp.cache.Get(sni)
	if ok {
		return nil
	}

	certPath, err := cp.cg.CertGenerate(sni)
	if err != nil {
		return fmt.Errorf("certificate provider: %w", err)
	}

	err = cp.LoadFile(certPath, cp.conf.CertKey)
	if err != nil {
		return fmt.Errorf("certificate provider: %w", err)
	}
	return nil
}

func (cp *CertProvider) Certificates() []tls.Certificate {
	return cp.cache.Array()
}

type CertGenerator struct {
	conf config.TLS
}

func (cg *CertGenerator) CertGenerate(domain string) (string, error) {
	id, _ := uuid.NewV7()
	serialNumber := "0x" + hex.EncodeToString(id[:])
	certPath := filepath.Join(cg.conf.CertsDir, domain)

	cmd := exec.Command(
		cg.conf.CertScript,
		cg.conf.CACert,
		cg.conf.CAKey,
		certPath,
		domain,
		serialNumber,
		cg.conf.CertKey,
	)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("generate certificate: %w", err)
	}
	return certPath + ".crt", nil
}

type CertCache struct {
	certs map[string]*tls.Certificate
	cg    CertGenerator

	mu *sync.RWMutex
}

func NewCertCache() *CertCache {
	return &CertCache{
		certs: make(map[string]*tls.Certificate),
		mu:    &sync.RWMutex{},
	}
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

func (c *CertCache) Array() []tls.Certificate {
	certs := make([]tls.Certificate, 0, len(c.certs))
	c.mu.RLock()
	for _, v := range c.certs {
		certs = append(certs, *v)
	}
	c.mu.RUnlock()
	return certs
}
