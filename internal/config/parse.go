package config

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

func Parse(path string) (Config, error) {
	k := koanf.New(".")

	err := k.Load(file.Provider(path), yaml.Parser())
	if err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	var conf Config
	k.Unmarshal("", &conf)

	return conf, nil
}
