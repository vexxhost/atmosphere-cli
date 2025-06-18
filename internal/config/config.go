package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/vexxhost/atmosphere/pkg/helm"
)

var k = koanf.New(".")

var parserMap = map[string]koanf.Parser{
	".yaml": yaml.Parser(),
	".yml":  yaml.Parser(),
	".toml": toml.Parser(),
	".json": json.Parser(),
}

type Config struct {
	Components map[string]helm.ComponentConfig `koanf:",remain"`
}

func Load(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Debug("config file does not exist", "path", configFile)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(configFile))
	parser, ok := parserMap[ext]
	if !ok {
		return fmt.Errorf("unsupported config file format: %s", configFile)
	}

	if err := k.Load(file.Provider(configFile), parser); err != nil {
		return fmt.Errorf("failed to load config file %s: %w", configFile, err)
	}

	log.Info("loaded config file", "path", configFile)
	return nil
}

func GetHelmComponent(name string) (*helm.ComponentConfig, error) {
	override := &helm.ComponentConfig{}

	if !k.Exists(name) {
		return override, nil
	}

	if err := k.Unmarshal(name, override); err != nil {
		return nil, fmt.Errorf("failed to unmarshal component %q: %w", name, err)
	}

	return override, nil
}
