package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var config Config
var configPath string

type Config struct {
	WorkingDirectory string           `yaml:"workingDir,omitempty"`
	HttpServer       HttpServerConfig `yaml:"httpServer"`
	Routes           *[]Route         `yaml:"routes"`
	*Authorization   `yaml:"authorization"`
	*Credentials     `yaml:"credentials"`
	*YandexHome      `yaml:"yandexHome"`
	*YandexDialogs   `yaml:"yandexDialogs"`
	*ZwCmd           `yaml:"zwCmd"`
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
	Dir            string        `yaml:"dir,omitempty"`
	File           string        `yaml:"file,omitempty"`
	DirMode        os.FileMode   `yaml:"dirMode,omitempty"`
	FileMode       os.FileMode   `yaml:"fileMode,omitempty"`
	MaxSize        string        `yaml:"maxSize,omitempty"`
	MaxAge         string        `yaml:"maxAge,omitempty"` // seconds
	Backups        uint32        `yaml:"backups,omitempty"`
	BackupDays     uint32        `yaml:"backupDays,omitempty"`
	Archive        string        `yaml:"archive,omitempty"`
	MaxSizeBytes   int64         `yaml:"-"`
	MaxAgeDuration time.Duration `yaml:"-"`
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

type Route struct {
	Type        string `yaml:"type"`
	Path        string `yaml:"path,omitempty"`
	RateLimit   string `yaml:"rateLimit,omitempty"`
	MaxBodySize string `yaml:"maxBodySize,omitempty"`
	Methods     string `yaml:"methods,omitempty"`
}

type Authorization struct {
	TokenSecret string                 `yaml:"tokenSecret,omitempty"`
	LifeTime    *AuthorizationLifeTime `yaml:"lifeTime,omitempty"`
}

type AuthorizationLifeTime struct {
	CodeToken    string `yaml:"codeToken,omitempty"`
	AccessToken  string `yaml:"accessToken,omitempty"`
	RefreshToken string `yaml:"refreshToken,omitempty"`
}

type Credentials struct {
	Users   []User   `yaml:"users"`
	Clients []Client `yaml:"clients,omitempty"`
}

type User struct {
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
	Scope    string `yaml:"scope,omitempty"`
}

type Client struct {
	Id          string   `yaml:"id"`
	Name        string   `yaml:"name,omitempty"`
	Secret      string   `yaml:"secret"`
	RedirectUri []string `yaml:"redirectUri,omitempty"`
	Options     string   `yaml:"options"`
	Scope       string   `yaml:"scope,omitempty"`
}

type YandexHome struct {
	Devices []YandexHomeDeviceConfig `yaml:"devices,omitempty"`
}

type YandexHomeDeviceConfig struct {
	Id           string                       `yaml:"id"`
	Name         string                       `yaml:"name"`
	Description  string                       `yaml:"description,omitempty"`
	Room         string                       `yaml:"room,omitempty"`
	Type         string                       `yaml:"type"`
	ZwId         byte                         `yaml:"zwid"`
	Capabilities []YandexHomeCapabilityConfig `yaml:"capabilities,omitempty"`
}

type YandexHomeCapabilityConfig struct {
	Retrievable bool
	Parameters  interface{}
}

func (c *YandexHomeCapabilityConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v map[string]bool
	unmarshal(&v)
	var ok bool
	if c.Retrievable, ok = v["on_off"]; ok {
		c.Parameters = YandexHomeParametersOnOff{}
	} else if c.Retrievable, ok = v["mode"]; ok {
		var p YandexHomeParametersModeConfig
		if err := unmarshal(&p); err != nil {
			return err
		}
		c.Parameters = p
	} else if c.Retrievable, ok = v["range"]; ok {
		var p YandexHomeParametersRangeConfig
		if err := unmarshal(&p); err != nil {
			return err
		}
		c.Parameters = p
	}
	return nil
}

type YandexHomeParametersOnOff struct {
}

type YandexHomeParametersModeConfig struct {
	Instance string   `yaml:"instance"`
	Values   []string `yaml:"values"`
}

type YandexHomeParametersRangeConfig struct {
	Instance     string  `yaml:"instance"`
	Units        string  `yaml:"units,omitempty"`
	RandomAccess *bool   `yaml:"randomAccess,omitempty"`
	Min          float64 `yaml:"min,omitempty"`
	Max          float64 `yaml:"max,omitempty"`
	Precision    float64 `yaml:"precision,omitempty"`
}

type YandexDialogs struct {
	Tales string `yaml:"tales,omitempty"`
}

type ZwCmd struct {
	Path    string `yaml:"path,omitempty"`
	Timeout int    `yaml:"timeout,omitempty"`
}

type configError func(msg string)

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
	ce := func(msg string) {
		errStr.WriteString(NewLine + msg)
	}

	configPath = filepath.Dir(cfgFile)

	if config.WorkingDirectory != "" {
		if err := os.Chdir(config.WorkingDirectory); err != nil {
			return fmt.Errorf("Unable to change working directory: %v", err)
		}
	}

	validate := []func(cfgError configError){
		validateHttpServerConfig,
		validateRouteConfig,
		validateCredentialsConfig,
		validateAuthorizationConfig,
		validateYandexHomeConfig,
		validateZwCmdConfig,
		validateYandexDialogsTalesConfig,
	}
	for _, v := range validate {
		v(ce)
	}
	if errStr.Len() > 0 {
		return fmt.Errorf("The configuration file is invalid:%v", errStr.String())
	}
	return nil
}

func loadSubConfig(subCfgFile string, cfg interface{}) error {
	if !filepath.IsAbs(subCfgFile) {
		subCfgFile = filepath.Join(configPath, subCfgFile)
	}
	file, err := os.Open(subCfgFile)
	if err != nil {
		return err
	}
	err = yaml.NewDecoder(file).Decode(cfg)
	if err != nil {
		return err
	}
	return nil
}
