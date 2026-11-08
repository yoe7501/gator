package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DBURL    string `json:"db_url"`
	UserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	var c Config
	path, err := getConfigFile()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}

func (c *Config) SetUser(name string) error {
	c.UserName = name
	path, err := getConfigFile()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0o644)
	if err != nil {
		return err
	}
	return nil
}

func getConfigFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".gatorconfig.json")
	return path, nil
}
