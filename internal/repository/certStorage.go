package repository

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
)

type CertStorage struct {
	mu              *sync.RWMutex
	certificates    map[string]tls.Certificate
	certificatesDir string
}

func NewCertStorage() *CertStorage {
	return &CertStorage{
		mu:           &sync.RWMutex{},
		certificates: make(map[string]tls.Certificate),
	}
}

func (cs *CertStorage) SetCertificatesDir(dir string) {
	cs.certificatesDir = dir
}

func (cs *CertStorage) GetOrCreateCertificate(domain string) (*tls.Certificate, error) {
	cs.mu.RLock()
	cert, ok := cs.certificates[domain]
	cs.mu.RUnlock()
	if ok {
		log.Printf("Выгружен сертификат для %s\n", domain)
		return &cert, nil
	}

	certFile, keyFile, err := GenerateCert(cs.certificatesDir, domain)
	if err != nil {
		return nil, err
	}

	cert, err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	cs.mu.Lock()
	cs.certificates[domain] = cert
	cs.mu.Unlock()

	log.Printf("Создан сертификат для %s\n", domain)
	return &cert, nil
}

func (cs *CertStorage) LoadCertificates(dir string) error {
	files, err := getSortedFilesInDir(dir)
	if err != nil {
		return err
	}

	certFiles := make(map[string]*struct {
		cert string
		key  string
	})

	for _, file := range files {
		fileBase := filepath.Base(file)
		if certFiles[fileBase] == nil {
			certFiles[fileBase] = new(struct{cert string; key string})
		}
		switch filepath.Ext(fileBase) {
		case ".crt":
			certFiles[fileBase].cert = file
		case ".key":
			certFiles[fileBase].key = file
		}
	}

	for domain, files := range certFiles {
		if files.cert != "" && files.key != "" {
			cert, err := tls.LoadX509KeyPair(files.cert, files.key)
			if err != nil {
				return err
			}
			cs.mu.Lock()
			cs.certificates[domain] = cert
			cs.mu.Unlock()
		}
	}

	return nil
}

func GenerateCert(certDir, domain string) (string, string, error) {
	keyFile := certDir + "/" + domain + ".key"
	csrFile := certDir + "/" + domain + ".csr"
	certFile := certDir + "/" + domain + ".crt"
	confFile := certDir + "/" + domain + ".conf"

	if certDir != "" {
		err := os.MkdirAll(certDir, 0755)
		if err != nil {
			return "", "", fmt.Errorf("не удалось создать директорию: %v", err)
		}
	}

	// Шаблон конфига
	conf := fmt.Sprintf(`
[req]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[dn]
CN = %s

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = %s
DNS.2 = www.%s
`, domain, domain, domain)

	err := os.WriteFile(confFile, []byte(conf), 0644)
	if err != nil {
		return "", "", fmt.Errorf("не удалось записать конфиг: %v", err)
	}

	// key generation
	if err := exec.Command("openssl", "genrsa", "-out", keyFile, "2048").Run(); err != nil {
		return "", "", fmt.Errorf("ошибка генерации ключа: %v", err)
	}

	// Генерация CSR
	if err := exec.Command("openssl", "req",
		"-new",
		"-key", keyFile,
		"-out", csrFile,
		"-config", confFile,
	).Run(); err != nil {
		return "", "", fmt.Errorf("ошибка генерации csr: %v", err)
	}

	// Signing cert
	if err := exec.Command("openssl", "x509", "-req",
		"-in", csrFile,
		"-CA", "ca.crt",
		"-CAkey", "ca.key",
		"-CAcreateserial",
		"-out", certFile,
		"-days", "3650",
		"-sha256",
		"-extensions", "req_ext",
		"-extfile", confFile,
	).Run(); err != nil {
		return "", "", fmt.Errorf("ошибка подписи: %v", err)
	}

	os.Remove(confFile)
	os.Remove(csrFile)

	return certFile, keyFile, nil
}

func getSortedFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	sort.StringSlice(files).Sort()

	return files, nil
}
