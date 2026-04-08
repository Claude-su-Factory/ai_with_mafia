package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   ServerConfig    `toml:"server"`
	Database DatabaseConfig  `toml:"database"`
	Redis    RedisConfig     `toml:"redis"`
	AI       AIConfig        `toml:"ai"`
	Game     GameConfig      `toml:"game"`
	Personas []PersonaConfig `toml:"personas"`
}

type ServerConfig struct {
	Port               int `toml:"port"`
	ReconnectGraceSec  int `toml:"reconnect_grace_sec"`
}

type DatabaseConfig struct {
	DSN string `toml:"dsn"`
}

type RedisConfig struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type AIConfig struct {
	APIKey            string `toml:"api_key"`
	ModelDefault      string `toml:"model_default"`
	ModelReasoning    string `toml:"model_reasoning"`
	MaxConcurrent     int    `toml:"max_concurrent"`
	HistoryMax        int    `toml:"history_max"`
	ResponseDelayMin  int    `toml:"response_delay_min"`
	ResponseDelayMax  int    `toml:"response_delay_max"`
}

type GameConfig struct {
	Mafia MafiaGameConfig `toml:"mafia"`
}

type MafiaGameConfig struct {
	Timers MafiaTimers `toml:"timers"`
}

type MafiaTimers struct {
	DayDiscussion int `toml:"day_discussion"`
	DayVote       int `toml:"day_vote"`
	Night         int `toml:"night"`
}

type PersonaConfig struct {
	Name        string `toml:"name"`
	Personality string `toml:"personality"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
