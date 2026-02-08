package config

type Config struct {
	Proxy   Proxy
	Panel   Panel
	Storage Storage
}

type Panel struct {
	Address string
	Port    string
}

type Storage struct {
	InMemory bool `koanf:"in_memory"`
}
