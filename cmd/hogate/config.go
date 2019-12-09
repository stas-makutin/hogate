package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var config Config

type Config struct {
	HttpServer HttpServerConfig `yaml:"httpServer"`
}

type HttpServerConfig struct {
	Port              int            `yaml:"port"`
	MaxConnections    uint           `yaml:"maxConnections,omitempty"`
	ReadTimeout       uint           `yaml:"readTimeout,omitempty"`       // milliseconds
	ReadHeaderTimeout uint           `yaml:"readHeaderTimeout,omitempty"` // milliseconds
	WriteTimeout      uint           `yaml:"writeTimeout,omitempty"`      // milliseconds
	IdleTimeout       uint           `yaml:"idleTimeout,omitempty"`       // milliseconds
	MaxHeaderBytes    uint32         `yaml:"maxHeaderBytes,omitempty"`
	Log               *HttpServerLog `yaml:"log,omitempty"`
	*TLSFiles         `yaml:"tlsFiles,omitempty"`
	*TLSAcme          `yaml:"tlsAcme,omitempty"`
}

type HttpServerLog struct {
	Dir      string      `yaml:"dir,omitempty"`
	File     string      `yaml:"file,omitempty"`
	FileMode os.FileMode `yaml:"fileMode,omitempty"`
	MaxSize  string      `yaml:"maxSize,omitempty"`
	MaxAge   string      `yaml:"maxAge,omitempty"` // seconds
	Backups  uint32      `yaml:"backups,omitempty"`
}

type TLSFiles struct {
	Certificate string `yaml:"certificate"`
	Key         string `yaml:"key"`
}

type TLSAcme struct {
	Email         string   `yaml:"email,omitempty"`        // contact email address
	HostWhitelist []string `yaml:"hostWhitelist"`          // allowed host names
	RenewBefore   uint32   `yaml:"renewBefore,omitempty"`  // renew days before expiration, default is 30 days
	CacheDir      string   `yaml:"cacheDir"`               // path to the directory
	DirectoryURL  string   `yaml:"directoryUrl,omitempty"` // ACME directory URL, default is Let's Encrypt directory
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
	var errStr strings.Builder
	validate := []func(sb strings.Builder){
		validateHttpServerConfig,
	}
	for _, v := range validate {
		v(errStr)
	}
	if errStr.Len() > 0 {
		return fmt.Errorf("The configuration file is invalid:%v", errStr)
	}
	return nil
}

func validateHttpServerConfig(errStr strings.Builder) {
	if config.HttpServer.Port < 1 || config.HttpServer.Port > 65535 {
		errStr.WriteString(NewLine + "httpServer.port must be between 1 and 65535.")
	}

	if config.HttpServer.Log != nil {
		if config.HttpServer.Log.Dir == "" {
			errStr.WriteString(NewLine + "httpServer.log.dir is required.")
		}
		if config.HttpServer.Log.File == "" {
			config.HttpServer.Log.File = appName + ".log"
		}
		if config.HttpServer.Log.FileMode == 0 {
			config.HttpServer.Log.FileMode = 0644
		}
		err := os.MkdirAll(config.HttpServer.Log.Dir, config.HttpServer.Log.FileMode)
		if err != nil {
			errStr.WriteString(NewLine + "httpServer.log.dir is not valid.")
		}
	}

	if config.HttpServer.TLSFiles != nil {
		if config.HttpServer.TLSFiles.Certificate == "" {
			errStr.WriteString(NewLine + "httpServer.TLSFiles.certificate must be specified.")
		} else if _, err := os.Stat(config.HttpServer.TLSFiles.Certificate); err != nil {
			errStr.WriteString(fmt.Sprintf("%vUnable to access the file using httpServer.TLSFiles.certificate path: %v", NewLine, err))
		}
		if config.HttpServer.TLSFiles.Key == "" {
			errStr.WriteString(NewLine + "httpServer.TLSFiles.key must be specified.")
		} else if _, err := os.Stat(config.HttpServer.TLSFiles.Key); err != nil {
			errStr.WriteString(fmt.Sprintf("%vUnable to access the file using httpServer.TLSFiles.key path: %v", NewLine, err))
		}
	}

	if config.HttpServer.TLSAcme != nil {
		if len(config.HttpServer.TLSAcme.HostWhitelist) <= 0 {
			errStr.WriteString(NewLine + "httpServer.TLSAcme.hostWhitelist must not be empty.")
		} else {
			for _, v := range config.HttpServer.TLSAcme.HostWhitelist {
				if v == "" {
					errStr.WriteString(NewLine + "httpServer.TLSAcme.hostWhitelist must not contain empty item.")
					break
				}
			}
		}
		if config.HttpServer.TLSAcme.CacheDir == "" {
			errStr.WriteString(NewLine + "httpServer.TLSAcme.cacheDir cannot be empty.")
		}
	}
}
