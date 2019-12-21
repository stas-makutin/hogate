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

func validateHttpServerConfig(cfgError configError) {
	if config.HttpServer.Port < 1 || config.HttpServer.Port > 65535 {
		cfgError("httpServer.port must be between 1 and 65535.")
	}

	if config.HttpServer.Log != nil {
		if config.HttpServer.Log.Dir == "" {
			cfgError("httpServer.log.dir is required.")
		}
		if config.HttpServer.Log.File == "" {
			config.HttpServer.Log.File = appName + ".log"
		}
		if config.HttpServer.Log.DirMode == 0 {
			config.HttpServer.Log.DirMode = 0755
		}
		if config.HttpServer.Log.FileMode == 0 {
			config.HttpServer.Log.FileMode = 0644
		}
		err := os.MkdirAll(config.HttpServer.Log.Dir, config.HttpServer.Log.DirMode)
		if err != nil {
			cfgError("httpServer.log.dir is not valid.")
		}

		size, err := parseSizeString(config.HttpServer.Log.MaxSize)
		if err == nil && size < 0 {
			err = fmt.Errorf("negative value not allowed.")
		}
		if err != nil {
			cfgError(fmt.Sprintf("httpServer.log.maxSize is not valid: %v", err))
		}
		config.HttpServer.Log.MaxSizeBytes = size

		duration, err := parseTimeDuration(config.HttpServer.Log.MaxAge)
		if err == nil && duration < 0 {
			err = fmt.Errorf("negative value not allowed.")
		}
		if err != nil {
			cfgError(fmt.Sprintf("httpServer.log.maxAge is not valid: %v", err))
		}
		config.HttpServer.Log.MaxAgeDuration = duration

		config.HttpServer.Log.Archive = strings.ToLower(config.HttpServer.Log.Archive)
		if !(config.HttpServer.Log.Archive == "" || config.HttpServer.Log.Archive == "zip") {
			cfgError("httpServer.log.archive could be either empty or has \"zip\" value")
		}
	}

	if config.HttpServer.TLSFiles != nil {
		if config.HttpServer.TLSFiles.Certificate == "" {
			cfgError("httpServer.TLSFiles.certificate must be specified.")
		} else if _, err := os.Stat(config.HttpServer.TLSFiles.Certificate); err != nil {
			cfgError(fmt.Sprintf("Unable to access the file using httpServer.TLSFiles.certificate path: %v", err))
		}
		if config.HttpServer.TLSFiles.Key == "" {
			cfgError("httpServer.TLSFiles.key must be specified.")
		} else if _, err := os.Stat(config.HttpServer.TLSFiles.Key); err != nil {
			cfgError(fmt.Sprintf("Unable to access the file using httpServer.TLSFiles.key path: %v", err))
		}
	}

	if config.HttpServer.TLSAcme != nil {
		if len(config.HttpServer.TLSAcme.HostWhitelist) <= 0 {
			cfgError("httpServer.TLSAcme.hostWhitelist must not be empty.")
		} else {
			for _, v := range config.HttpServer.TLSAcme.HostWhitelist {
				if v == "" {
					cfgError("httpServer.TLSAcme.hostWhitelist must not contain empty item.")
					break
				}
			}
		}
		if config.HttpServer.TLSAcme.CacheDir == "" {
			cfgError("httpServer.TLSAcme.cacheDir cannot be empty.")
		}
	}
}

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

	addOAuthRoutes(router)

	var handler http.Handler = router
	if config.HttpServer.Log != nil {
		handler = logHandler(errorLog)(handler)
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

func logHandler(errorLog *log.Logger) func(http.Handler) http.Handler {
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
