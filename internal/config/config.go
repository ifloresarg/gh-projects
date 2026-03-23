package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultOwner    string `yaml:"default_owner"`
	DefaultProject  int    `yaml:"default_project"`
	CacheTTL        int    `yaml:"cache_ttl"`
	ShowLabels      bool   `yaml:"show_labels"`
	ShowClosedItems bool   `yaml:"show_closed_items"`
	MergedPRWindow  int    `yaml:"merged_pr_window"`
	PRFetchLimit    int    `yaml:"pr_fetch_limit"`
}

// configPathOverride is used for testing to override the default config path.
var configPathOverride string

func DefaultConfig() Config {
	return Config{
		DefaultOwner:    "",
		DefaultProject:  0,
		CacheTTL:        300,
		ShowLabels:      true,
		ShowClosedItems: false,
		MergedPRWindow:  12,
		PRFetchLimit:    200,
	}
}

func configPath() (string, error) {
	if configPathOverride != "" {
		return configPathOverride, nil
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh-projects", "config.yaml"), nil
}

// Load reads config from ~/.config/gh-projects/config.yaml.
// Returns defaults if file doesn't exist.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, err
	}

	cfg := DefaultConfig()
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save writes config to ~/.config/gh-projects/config.yaml, creating dirs as needed.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
