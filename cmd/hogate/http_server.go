package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/netutil"
)

type httpServer struct {
	server *http.Server
}

func (srv *httpServer) start(errorLog *log.Logger) error {
	useTLS := false
	tlsCertFile := ""
	tlsKeyFile := ""
	var acmeManager *autocert.Manager
	var tlsConfig *tls.Config

	if config.HttpServer.TLSFiles != nil {
		tlsCertFile = config.HttpServer.TLSFiles.Certificate
		tlsKeyFile = config.HttpServer.TLSFiles.Key
	} else if config.HttpServer.TLSAcme != nil {
		var acmeClient *acme.Client
		if config.HttpServer.TLSAcme.DirectoryURL != "" {
			acmeClient = &acme.Client{
				DirectoryURL: config.HttpServer.TLSAcme.DirectoryURL,
			}
		}
		acmeManager = &autocert.Manager{
			Cache:       autocert.DirCache(config.HttpServer.TLSAcme.CacheDir),
			Prompt:      autocert.AcceptTOS,
			HostPolicy:  autocert.HostWhitelist(config.HttpServer.TLSAcme.HostWhitelist...),
			RenewBefore: time.Duration(config.HttpServer.TLSAcme.RenewBefore) * time.Hour * 24,
			Email:       config.HttpServer.TLSAcme.Email,
			Client:      acmeClient,
		}
		tlsConfig = acmeManager.TLSConfig()
	}

	// create TCP listener
	netListener, err := net.Listen("tcp", ":"+strconv.Itoa(int(config.HttpServer.Port)))
	if err != nil {
		return fmt.Errorf("Listen on %v port failed: %v", config.HttpServer.Port, err)
	}
	defer netListener.Close()

	// apply concurrent connections limit
	if config.HttpServer.MaxConnections > 0 {
		netListener = netutil.LimitListener(netListener, config.HttpServer.MaxConnections)
	}

	srv.server = &http.Server{
		//Handler:      router,
		ReadTimeout:       time.Millisecond * time.Duration(config.HttpServer.ReadTimeout),
		ReadHeaderTimeout: time.Millisecond * time.Duration(config.HttpServer.ReadHeaderTimeout),
		WriteTimeout:      time.Millisecond * time.Duration(config.HttpServer.WriteTimeout),
		IdleTimeout:       time.Millisecond * time.Duration(config.HttpServer.IdleTimeout),
		MaxHeaderBytes:    config.HttpServer.MaxHeaderBytes,
		ErrorLog:          errorLog,
		TLSConfig:         tlsConfig,
	}

	if useTLS {
		err = srv.server.ServeTLS(netListener, tlsCertFile, tlsKeyFile)
	} else {
		err = srv.server.Serve(netListener)
	}

	// err = srv.server.ServeTLS(netListener, "../../cert.pem", "../../cert.key")
	if err != nil {
		return fmt.Errorf("Failed to start HTTP server: %v", err)
	}

	return nil
}

func (srv *httpServer) stop() error {
	return nil
}
