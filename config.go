package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server   ServerConfig `toml:"server" json:"server"`
	VPSList  []VPSConfig  `toml:"vps_list" json:"vps_list"`
	Alert    AlertConfig  `toml:"alert" json:"alert"`
	Telegram TGConfig     `toml:"telegram" json:"-"` // Added to TOML, hidden in JSON
	Password string       `json:"-"`                 // Loaded from .env
}

type AlertConfig struct {
	Threshold          float64 `toml:"threshold" json:"threshold"`
	ConsecutiveCount   int     `toml:"consecutive_count" json:"consecutive_count"`
	CooldownMinutes    int     `toml:"cooldown_minutes" json:"cooldown_minutes"`
}

type TGConfig struct {
	Token  string `toml:"token"`
	ChatID string `toml:"chat_id"`
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

	// Allow environment variables to override TOML config
	if envToken := os.Getenv("TG_BOT_TOKEN"); envToken != "" {
		cfg.Telegram.Token = envToken
	}
	if envChatID := os.Getenv("TG_CHAT_ID"); envChatID != "" {
		cfg.Telegram.ChatID = envChatID
	}

	return &cfg, nil
}
