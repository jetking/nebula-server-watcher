package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server  ServerConfig `toml:"server" json:"server"`
	VPSList []VPSConfig  `toml:"vps_list" json:"vps_list"`
	Password string      `json:"-"` // Not in TOML, loaded from .env
}

type ServerConfig struct {
	Port int `toml:"port" json:"port"`
}

type VPSConfig struct {
	ID      string `toml:"id" json:"id"`
	Name    string `toml:"name" json:"name"`
	IP      string `toml:"ip" json:"ip"`
	Country string `toml:"country" json:"country"`
	Remarks string `toml:"remarks" json:"remarks"`
}

func LoadConfig(path string) (*Config, error) {
	// 1. Load TOML
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// 2. Load .env
	_ = godotenv.Load() // Ignore error if .env doesn't exist
	cfg.Password = os.Getenv("WATCHER_PASSWORD")
	if cfg.Password == "" {
		cfg.Password = "change_me" // Default
	}

	return &cfg, nil
}
