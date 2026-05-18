// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

const (
	// redactedPassword is the placeholder shown in List output for the password field.
	redactedPassword = "********"
	// filePermissions is the Unix file mode for the config file (owner read/write only).
	filePermissions = 0600
)

// Store manages reading and writing the cz-otel configuration file.
type Store struct {
	path string
}

// NewStore creates a new Store. If path is empty, the default config file path is used.
func NewStore(path string) *Store {
	if path == "" {
		path = paths.ConfigFilePath()
	}
	return &Store{path: path}
}

// Load reads the configuration from disk and returns it as a map.
// If the file does not exist, an empty map is returned.
func (s *Store) Load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg map[string]string
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	if cfg == nil {
		cfg = make(map[string]string)
	}
	return cfg, nil
}

// Save writes the configuration map to disk with restricted permissions (0600).
func (s *Store) Save(data map[string]string) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(s.path, out, filePermissions); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// Get retrieves the value for a configuration key. If the key is not explicitly
// set, it returns the default value. If no default exists, it returns an error.
func (s *Store) Get(key string) (string, error) {
	if !IsValidKey(key) {
		return "", fmt.Errorf("unrecognized config key: %q", key)
	}

	cfg, err := s.Load()
	if err != nil {
		return "", err
	}

	if val, ok := cfg[key]; ok {
		return val, nil
	}

	if def := DefaultValue(key); def != "" {
		return def, nil
	}

	return "", fmt.Errorf("config key %q is not set", key)
}

// Set validates the key, then loads the config, sets the value, and saves.
func (s *Store) Set(key, value string) error {
	if !IsValidKey(key) {
		return fmt.Errorf("unrecognized config key: %q — valid keys: %v", key, ValidKeys)
	}

	cfg, err := s.Load()
	if err != nil {
		return err
	}

	cfg[key] = value
	return s.Save(cfg)
}

// List returns all configuration values with the password field redacted.
func (s *Store) List() (map[string]string, error) {
	cfg, err := s.Load()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(cfg))
	for k, v := range cfg {
		if k == "password" {
			result[k] = redactedPassword
		} else {
			result[k] = v
		}
	}
	return result, nil
}
