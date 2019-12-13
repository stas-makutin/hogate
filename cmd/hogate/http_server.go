package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/netutil"
)

type httpServer struct {
	server *http.Server
	lock   sync.Mutex
}

const httpLogMessage = "logMessage"

func (srv *httpServer) init(errorLog *log.Logger) (useTLS bool, tlsCertFile, tlsKeyFile string) {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	var acmeManager *autocert.Manager
	var tlsConfig *tls.Config

	if config.HttpServer.TLSFiles != nil {
		tlsCertFile = config.HttpServer.TLSFiles.Certificate
		tlsKeyFile = config.HttpServer.TLSFiles.Key
		useTLS = true
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
		useTLS = true
	}

	router := http.NewServeMux()

	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "test")
	}))

	var handler http.Handler = router
	if config.HttpServer.Log != nil {
		handler = logMiddleware(errorLog)(handler)
	}

	srv.server = &http.Server{
		Handler:           handler,
		ReadTimeout:       time.Millisecond * time.Duration(config.HttpServer.ReadTimeout),
		ReadHeaderTimeout: time.Millisecond * time.Duration(config.HttpServer.ReadHeaderTimeout),
		WriteTimeout:      time.Millisecond * time.Duration(config.HttpServer.WriteTimeout),
		IdleTimeout:       time.Millisecond * time.Duration(config.HttpServer.IdleTimeout),
		MaxHeaderBytes:    int(config.HttpServer.MaxHeaderBytes),
		ErrorLog:          errorLog,
		TLSConfig:         tlsConfig,
	}

	return
}

func (srv *httpServer) start(errorLog *log.Logger) error {

	useTLS, tlsCertFile, tlsKeyFile := srv.init(errorLog)

	// create TCP listener
	netListener, err := net.Listen("tcp", ":"+strconv.Itoa(int(config.HttpServer.Port)))
	if err != nil {
		return fmt.Errorf("Listen on %v port failed: %v", config.HttpServer.Port, err)
	}
	defer netListener.Close()

	// apply concurrent connections limit
	if config.HttpServer.MaxConnections > 0 {
		netListener = netutil.LimitListener(netListener, int(config.HttpServer.MaxConnections))
	}

	if useTLS {
		err = srv.server.ServeTLS(netListener, tlsCertFile, tlsKeyFile)
	} else {
		err = srv.server.Serve(netListener)
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("Failed to start HTTP server: %v", err)
	}
	return nil
}

func (srv *httpServer) stop() error {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	var err error
	if srv.server != nil {
		err = srv.server.Shutdown(context.Background())
	}

	if err != nil {
		return fmt.Errorf("Error occured during HTTP server stop: %v", err)
	}
	return nil
}

type logResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	contentLength int64
}

func (w *logResponseWriter) WriteHeader(status int) {
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *logResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.contentLength += int64(n)
	return n, err
}

func httpAppendToLog(r *http.Request, message string) {
	if builder, ok := r.Context().Value(httpLogMessage).(*strings.Builder); ok {
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(message)
	}
}

func logMiddleware(errorLog *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().Local()
			lrw := &logResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			var bld strings.Builder
			ctx := context.WithValue(r.Context(), httpLogMessage, &bld)
			defer func() {
				record := []string{
					start.Format("2006-01-02T15:04:05.999"),
					strconv.FormatInt(int64(time.Now().Local().Sub(start)/time.Millisecond), 10),
					r.RemoteAddr,
					r.Host,
					r.Proto,
					r.Method,
					r.RequestURI,
					strconv.FormatInt(r.ContentLength, 10),
					r.Header.Get("X-Request-Id"),
					strconv.Itoa(lrw.statusCode),
					strconv.FormatInt(lrw.contentLength, 10),
					bld.String(),
				}

				var b bytes.Buffer
				csvw := csv.NewWriter(&b)
				csvw.Write(record)
				csvw.Flush()

				logCfg := config.HttpServer.Log
				logFile := filepath.Join(logCfg.Dir, logCfg.File)

				logRotate.rotate(logFile, errorLog)

				var f *os.File
				var err error
				for i := 0; i < 6; i++ {
					f, err = os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, logCfg.FileMode)
					if err == nil {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
				if err == nil {
					defer f.Close()
					_, err = f.Write(b.Bytes())
				}
				if err != nil {
					errorLog.Printf("Unable to log HTTP request:%v%v%vreason: %v", NewLine, string(b.Bytes()), NewLine, err)
				}
			}()
			next.ServeHTTP(lrw, r.WithContext(ctx))
		})
	}
}
