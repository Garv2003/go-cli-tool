package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	URLPattern string   `json:"urlPattern"`
	EnvNames   []string `json:"envNames"`
}

func (c *Config) ReloadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(c)
}
