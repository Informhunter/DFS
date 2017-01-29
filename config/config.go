package config

import (
	"encoding/json"
	"os"
)

type NodeInfo struct {
	Name           string
	PublicAddress  string
	PrivateAddress string
}

type Config struct {
	fileName string
	This     NodeInfo
	Nodes    []NodeInfo
}

func (config *Config) Load(configFileName string) error {
	configFile, err := os.Open(configFileName)
	if err != nil {
		return err
	}
	defer configFile.Close()

	dec := json.NewDecoder(configFile)
	err = dec.Decode(config)
	if err != nil {
		return err
	}
	config.fileName = configFileName
	return nil
}

func (config Config) Save() {
	config.SaveAs(config.fileName)
}

func (config Config) SaveAs(configFileName string) error {
	configFile, err := os.Create(configFileName)
	if err != nil {
		return err
	}
	defer configFile.Close()

	enc := json.NewEncoder(configFile)
	enc.SetIndent("", "  ")
	err = enc.Encode(config)
	if err != nil {
		return err
	}
	return nil
}
