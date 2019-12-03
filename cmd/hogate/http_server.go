package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/netutil"
)

type httpServer struct {
	server *http.Server
}

func (srv *httpServer) start(errorLog *log.Logger) error {
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
	}

	err = srv.server.Serve(netListener)
	// err = srv.server.ServeTLS(netListener, "../../cert.pem", "../../cert.key")
	if err != nil {
		return fmt.Errorf("Failed to start HTTP server: %v", err)
	}

	return nil
}

func (srv *httpServer) stop() error {
	return nil
}
