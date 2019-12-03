package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var config Config

type Config struct {
	HttpServer HttpServerConfig `yaml:"httpServer"`
}

type HttpServerConfig struct {
	Port              int `yaml:"port"`
	MaxConnections    int `yaml:"maxConnections,omitempty"`
	ReadTimeout       int `yaml:"readTimeout,omitempty"`       // milliseconds
	ReadHeaderTimeout int `yaml:"readHeaderTimeout,omitempty"` // milliseconds
	WriteTimeout      int `yaml:"writeTimeout,omitempty"`      // milliseconds
	IdleTimeout       int `yaml:"idleTimeout,omitempty"`       // milliseconds
	MaxHeaderBytes    int `yaml:"maxHeaderBytes,omitempty"`
	TLSFiles          `yaml:"tlsFiles,omitempty"`
	TLSAcme           `yaml:"tlsAcme,omitempty"`
}

type TLSFiles struct {
	Certificate string `yaml:"certificate,omitempty"`
	Key         string `yaml:"key,omitempty"`
}

type TLSAcme struct {
}

func loadConfig(cfgFile string) error {
	file, err := os.Open(cfgFile)
	if err != nil {
		return fmt.Errorf("Unable to open configuration file: %v", err)
	}
	err = yaml.NewDecoder(file).Decode(&config)
	if err != nil {
		return fmt.Errorf("Unable to parse configuration file: %v", err)
	}
	validate := []func() error{
		validateHttpServerConfig,
	}
	for _, v := range validate {
		err = v()
		if err != nil {
			return err
		}
	}

	return nil
}

func validateHttpServerConfig() error {
	if config.HttpServer.Port < 1 || config.HttpServer.Port > 65535 {
		return fmt.Errorf("httpServer.port must be between 1 and 65535.")
	}
	return nil
}
