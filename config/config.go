package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

const (
	configDirName  = "reap"
	configFileName = "config.yaml"
)

type Config struct {
	DefaultDepth int    `yaml:"default_depth"`
	Repos        []Repo `yaml:"repos"`
}

type Repo struct {
	URL      string  `yaml:"url"`
	Selected bool    `yaml:"selected"`
	Groups   []Group `yaml:"groups,omitempty"`
}

type Group struct {
	Name     string `yaml:"name"`
	Selected bool   `yaml:"selected"`
}

func GetConfigPath() (string, error) {
	var configHome string
	var err error

	if runtime.GOOS == "windows" {
		configHome, err = os.UserConfigDir()
		if err != nil {
			return "", err
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}

	configDirPath := filepath.Join(configHome, configDirName)
	if err := os.MkdirAll(configDirPath, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDirPath, configFileName), nil
}

func Load() (*Config, bool, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, false, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg, err := createDefaultConfig(path)
		return cfg, true, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, false, err
	}

	return &cfg, false, nil
}

func Save(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func createDefaultConfig(path string) (*Config, error) {
	cfg := &Config{
		Repos: []Repo{},
	}

	if err := Save(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) HasGroups() bool {
	for _, repo := range c.Repos {
		if len(repo.Groups) > 0 {
			return true
		}
	}
	return false
}
