package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigName = ".reap.yaml"
)

type Config struct {
	Repos []Repo `yaml:"repos"`
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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultConfigName), nil
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
