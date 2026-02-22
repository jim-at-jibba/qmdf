package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all qmdf configuration.
type Config struct {
	Collection   string  `mapstructure:"collection"`
	Mode         string  `mapstructure:"mode"`
	Results      int     `mapstructure:"results"`
	MinScore     float64 `mapstructure:"min_score"`
	NoPreview    bool    `mapstructure:"no_preview"`
	PreviewWidth float64 `mapstructure:"preview_width"` // fraction: 0.0–1.0
	Editor       string  `mapstructure:"editor"`
	PrintMode    bool    `mapstructure:"-"` // CLI flag only, not in config file
}

// Defaults returns a Config with sensible default values.
func Defaults() *Config {
	return &Config{
		Mode:         "search",
		Results:      10,
		MinScore:     0.0,
		NoPreview:    false,
		PreviewWidth: 0.55,
		Editor:       "",
	}
}

// Load reads config from ~/.config/qmdf/config.yaml, overlaying defaults.
// CLI flags applied afterwards (by cobra) take highest priority.
func Load() (*Config, error) {
	cfg := Defaults()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Config search paths
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		v.AddConfigPath(filepath.Join(xdg, "qmdf"))
	}
	home, _ := os.UserHomeDir()
	v.AddConfigPath(filepath.Join(home, ".config", "qmdf"))

	// Environment variable prefix: QMDF_COLLECTION, QMDF_MODE, etc.
	v.SetEnvPrefix("QMDF")
	v.AutomaticEnv()

	// Bind defaults so viper knows the key names
	v.SetDefault("collection", cfg.Collection)
	v.SetDefault("mode", cfg.Mode)
	v.SetDefault("results", cfg.Results)
	v.SetDefault("min_score", cfg.MinScore)
	v.SetDefault("no_preview", cfg.NoPreview)
	v.SetDefault("preview_width", cfg.PreviewWidth)
	v.SetDefault("editor", cfg.Editor)

	// Read config file (missing file is OK)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
