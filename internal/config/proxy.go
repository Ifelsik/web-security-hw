package config

type Proxy struct {
	Address string
	Port    string
	TLS     TLS `koanf:"tls"`
}

type TLS struct {
	CACert     string `koanf:"ca_cert"`
	CAKey      string `koanf:"ca_key"`
	CertsDir   string `koanf:"certs_dir"`
	CertKey    string `koanf:"cert_key"`
	CertScript string `konf:"cert_generator_script"`
}
