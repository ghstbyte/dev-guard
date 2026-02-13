package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type TrackerConfig struct {
	DailyTargetMinutes int    `yaml:"daily_target_minutes"` // Цель минут в день
	TrackerProcess     string `yaml:"tracked_process"`      // Отслеживаемый для работы процесс
}

type StrictModeConfig struct {
	Enabled            bool     `yaml:"enabled"`             // Строгий режим вкл\выкл
	ForbiddenProcesses []string `yaml:"forbidden_processes"` // Заблокированные процессы
}

type EnforcerConfig struct {
	StrictMode StrictModeConfig `yaml:"strict_mode"` // Конфиг строгого режима
}

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Tracker  TrackerConfig  `yaml:"tracker"`
	Enforcer EnforcerConfig `yaml:"enforcer"`
}

func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
