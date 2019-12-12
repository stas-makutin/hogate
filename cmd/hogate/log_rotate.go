package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type logRotation struct {
	lock uint32
}

var logRotate logRotation
var logRotatePattern = regexp.MustCompile(`^-(\d{4}-\d{2}-\d{2})_(\d+)$`)

func (r *logRotation) rotate(logFile string, errorLog *log.Logger) {
	logCfg := config.HttpServer.Log

	if !((logCfg.Backups > 0 || logCfg.BackupDays > 0) && (logCfg.MaxSizeBytes > 0 || logCfg.MaxAgeDuration > 0)) {
		return // rotation is not enabled
	}

	if atomic.SwapUint32(&r.lock, 1) != 0 {
		return // rotation in progress
	}

	rotate := false
	statusFile := logFile + ".status"

	fi, err := os.Stat(logFile)
	if err == nil {
		if logCfg.MaxAgeDuration > 0 {
			sfi, err := os.Stat(statusFile)
			if err != nil {
				if os.IsNotExist(err) {
					_, err = os.OpenFile(statusFile, os.O_CREATE, logCfg.FileMode)
				}
				if err != nil {
					errorLog.Printf("HTTP log rotation status file error: %v", err)
				}
			} else if time.Now().Sub(sfi.ModTime()) > logCfg.MaxAgeDuration {
				rotate = true
			}
		}
		if logCfg.MaxSizeBytes > 0 && fi.Size() > logCfg.MaxSizeBytes {
			rotate = true
		}
	}

	if rotate {
		go func() {
			defer atomic.SwapUint32(&r.lock, 0)

			var now time.Time

			// rename log file
			backupFile := logFile + ".backup"
			for i := 0; i < 6; i++ {
				now = time.Now()
				err = os.Rename(logFile, backupFile)
				if err == nil || os.IsNotExist(err) {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if err != nil {
				if !os.IsNotExist(err) {
					errorLog.Printf("HTTP log rotation backup file error: %v", err)
				}
				return
			}

			if logCfg.MaxAgeDuration > 0 {
				// touch status file
				// change it once https://github.com/golang/go/issues/32558 will be fixed
				defer func() {
					if err := os.Chtimes(statusFile, now, now); err != nil {
						errorLog.Printf("HTTP log rotation status file touch error: %v", err)
					}
				}()
			}

			// delete old backup files
			var files backupFiles
			files.populate(logFile, "", errorLog)

			// rename/archive backup file
		}()
	} else {
		atomic.SwapUint32(&r.lock, 0)
	}
}

type backupFileInfo struct {
	path    string
	date    string
	ordinal int
}

func (l *backupFileInfo) Less(r *backupFileInfo) bool {
	rc := strings.Compare(l.date, r.date)
	if rc > 0 {
		return true
	} else if rc == 0 {
		return l.ordinal > r.ordinal
	}
	return false
}

type backupFiles struct {
	files []backupFileInfo
}

func (f *backupFiles) Len() int {
	return len(f.files)
}

func (f *backupFiles) Swap(i, j int) {
	f.files[i], f.files[j] = f.files[j], f.files[i]
}

func (f *backupFiles) Less(i, j int) bool {
	return f.files[i].Less(&f.files[j])
}

func (f *backupFiles) populate(logFile, extension string, errorLog *log.Logger) {
	var errors strings.Builder

	logDir := filepath.Dir(logFile)
	err := filepath.Walk(logDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			errors.WriteString(NewLine + err.Error())
			return nil
		}
		if fi.IsDir() {
			if path == logDir {
				return nil
			}
			return filepath.SkipDir
		}
		if strings.HasPrefix(path, logFile) && strings.HasSuffix(path, extension) && path != logFile {
			name := path[len(logFile) : len(path)-len(extension)]
			m := logRotatePattern.FindStringSubmatch(name)
			if m != nil {
				if ordinal, err := strconv.Atoi(m[2]); err == nil {
					f.files = append(f.files, backupFileInfo{path: path, date: m[1], ordinal: ordinal})
				}
			}
		}
		return nil
	})

	if err != nil {
		errors.WriteString(NewLine + err.Error())
	}
	if errors.Len() > 0 {
		errorLog.Printf("HTTP log rotation - get backup files error:%v", errors)
		f.files = nil
	}
	f.files = nil
	sort.Sort(f)
}
