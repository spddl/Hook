package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LogPath string  `toml:"logPath"`
	Games   []Games `toml:"games"`
}

type Games struct {
	Exe             string    `toml:"exe"`
	Ioprio          int       `toml:"ioprio"`
	OnProcessStart  []Scripts `toml:"OnProcessStart,omitempty"`
	OnProcessFinish []Scripts `toml:"OnProcessFinish,omitempty"`
}

type Scripts struct {
	Name         string `toml:"name"`
	Args         string `toml:"args"`
	HideWindow   bool   `toml:"hideWindow,omitempty"`
	OnForeground bool   `toml:"onForeground,omitempty"`
	OnBackground bool   `toml:"onBackground,omitempty"`
}

func (c *Config) ReadConfig() {
	configPath := filepath.Join(executablePath, "config.toml")
	tomlData, err := os.ReadFile(configPath)
	if err != nil {
		println(err)
	}

	if _, err := toml.Decode(string(tomlData), &c); err != nil {
		println(err)
	}
}

func (c *Config) SaveConfig() {
	f, err := os.Create(filepath.Join(executablePath, "config.toml"))
	if err != nil {
		// failed to create/open the file
		log.Fatal(err)
	}
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		// failed to encode
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		// failed to close the file
		log.Fatal(err)
	}
}

func (c *Config) GetList() map[string]Games {
	g := map[string]Games{}

	for _, v := range c.Games {
		g[strings.ToLower(v.Exe)] = v
	}

	return g
}
