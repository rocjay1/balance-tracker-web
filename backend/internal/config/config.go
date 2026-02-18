package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Subscribers        []string     `yaml:"subscribers"`
	AlertDaysBeforeDue int          `yaml:"alert_days_before_due"`
	Cards              []CardConfig `yaml:"cards"`
	SMTP               SMTPConfig   `yaml:"smtp"`
	Timezone           string       `yaml:"timezone"`
}

type CardConfig struct {
	Name            string  `yaml:"name"`
	AccountNumber   string  `yaml:"account_number"` // Optional, for disambiguation
	Limit           float64 `yaml:"limit"`
	StatementDay    int     `yaml:"statement_day"`
	DueDay          int     `yaml:"due_day"`
	StartingBalance float64 `yaml:"starting_balance"` // Balance as of StartingDate
	StartingDate    string  `yaml:"starting_date"`    // YYYY-MM-DD
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Timezone == "" {
		cfg.Timezone = "America/Chicago"
	}

	if cfg.AlertDaysBeforeDue == 0 {
		cfg.AlertDaysBeforeDue = 3
	}

	return &cfg, nil
}
