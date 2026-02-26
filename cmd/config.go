package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type streepDefaults struct {
	Event    string `yaml:"event"`
	Job      string `yaml:"job"`
	Workflow string `yaml:"workflow"`
	DiffBase string `yaml:"diff_base"`
}

type streepConfig struct {
	Defaults     streepDefaults    `yaml:"defaults"`
	RunnerImages map[string]string `yaml:"runner_images"`
}

func loadStreepConfig(dir string) (streepConfig, error) {
	if dir == "" {
		dir = "."
	}
	path := filepath.Join(dir, ".streep", "config.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return streepConfig{}, nil
	}
	if err != nil {
		return streepConfig{}, fmt.Errorf("read config: %w", err)
	}

	var cfg streepConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return streepConfig{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
