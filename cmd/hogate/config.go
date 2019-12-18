package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var config Config

type Config struct {
	HttpServer     HttpServerConfig `yaml:"httpServer"`
	Routes         *[]Route         `yaml:"routes"`
	*Authorization `yaml:"authorization"`
	*Credentials   `yaml:"credentials"`
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
	Id          string `yaml:"id"`
	Name        string `yaml:"name,omitempty"`
	Secret      string `yaml:"secret"`
	RedirectUri string `yaml:"redirectUri,omitempty"`
	Options     string `yaml:"options"`
	Scope       string `yaml:"scope,omitempty"`
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
	validate := []func(cfgError configError){
		validateHttpServerConfig,
		validateRouteConfig,
		validateCredentialsConfig,
		validateAuthorizationConfig,
	}
	for _, v := range validate {
		v(ce)
	}
	if errStr.Len() > 0 {
		return fmt.Errorf("The configuration file is invalid:%v", errStr.String())
	}
	return nil
}
