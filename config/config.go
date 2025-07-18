package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"

	"github.com/joshjon/iot-metrics/log"
)

// Config holds application configuration loaded from environment variables
// and/or a YAML file.
type Config struct {
	Port            int        `yaml:"port" env:"PORT"`            // default: 8080
	SQLiteDir       string     `yaml:"sqliteDir" env:"SQLITE_DIR"` // default: ./data/
	Logger          Logger     `yaml:"logger" envPrefix:"LOGGER_"`
	DeviceRateLimit *RateLimit `yaml:"deviceRateLimit" envPrefix:"DEVICE_RATE_LIMIT_"`
}

func (c Config) Validate() []error {
	var errs []error
	if _, ok := log.ParseLevel(c.Logger.Level); !ok {
		errs = append(errs, errors.New("logger.level: must be one of [debug, info, warn, error]"))
	}
	if c.Port < 1 && c.Port > 65535 {
		errs = append(errs, errors.New("port: must be between 1 and 65535"))
	}
	if c.DeviceRateLimit != nil {
		if c.DeviceRateLimit.Tokens <= 0 {
			errs = append(errs, errors.New("deviceRateLimit.tokens: must be greater than 0"))
		}
		if c.DeviceRateLimit.Seconds <= 0 {
			errs = append(errs, errors.New("deviceRateLimit.seconds: must be greater than 0"))
		}
	}
	return errs
}

// Logger defines settings for application log output.
type Logger struct {
	Level      string `yaml:"level" env:"LEVEL"`           // default: info
	Structured bool   `yaml:"structured" env:"STRUCTURED"` // default: true
}

// RateLimit specifies token bucket parameters for rate limiting.
type RateLimit struct {
	// Maximum number of requests allowed per interval.
	Tokens int `yaml:"tokens" env:"TOKENS"`
	// Duration of the interval in seconds.
	Seconds int `yaml:"seconds" env:"SECONDS"`
}

// Load reads the application config from a YAML file and environment variables.
func Load(configFile string) (*Config, error) {
	cfg := Config{
		Port:      8080,
		SQLiteDir: "./data/",
		Logger: Logger{
			Level:      "info",
			Structured: true,
		},
	}

	if configFile != "" {
		file, err := os.Open(configFile)
		if err != nil {
			return nil, fmt.Errorf("open config file at path '%s': %w", configFile, err)
		}
		defer file.Close() //nolint:errcheck
		decoder := yaml.NewDecoder(file)
		if err = decoder.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("decode config file at path '%s': %w", configFile, err)
		}
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config environment variables: %w", err)
	}

	verrs := cfg.Validate()
	if len(verrs) > 0 {
		fmt.Fprintln(os.Stderr, "Config errors:")
		for _, verr := range verrs {
			if verr != nil {
				fmt.Fprintln(os.Stderr, "  "+verr.Error())
			}
		}
		fmt.Fprintln(os.Stdout) //nolint:errcheck
		os.Exit(1)
	}

	return &cfg, nil
}
