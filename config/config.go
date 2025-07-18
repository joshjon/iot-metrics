package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"

	"github.com/joshjon/iot-metrics/log"
)

type Config struct {
	Logger          Logger     `yaml:"logger" envPrefix:"LOGGER_"`
	Port            int        `yaml:"port" env:"PORT"`            // default: 8080
	SQLiteDir       string     `yaml:"sqliteDir" env:"SQLITE_DIR"` // default: ./data/
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

type Logger struct {
	Level      string `yaml:"level" env:"LEVEL"`           // default: info
	Structured bool   `yaml:"structured" env:"STRUCTURED"` // default: true
}

type RateLimit struct {
	Tokens  int `yaml:"tokens" env:"TOKENS"`
	Seconds int `yaml:"seconds" env:"SECONDS"`
}

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
		defer file.Close()
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
		fmt.Fprintln(os.Stdout)
		os.Exit(1)
	}

	return &cfg, nil
}
