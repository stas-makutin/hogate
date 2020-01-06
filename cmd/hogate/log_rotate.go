package main

import (
	"fmt"
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
var logRotatePattern = regexp.MustCompile(`^-(\d{4}-\d{2}-\d{2})(_(\d+))*$`)

func (r *logRotation) rotate(logFile string, errorLog *log.Logger) {
	logCfg := config.HttpServer.Log

	if !((logCfg.Backups > 0 || logCfg.BackupDays > 0) && (logCfg.MaxSizeBytes > 0 || logCfg.MaxAgeDuration > 0)) {
		return // rotation is not enabled
	}

	if atomic.SwapUint32(&r.lock, 1) != 0 {
		return // rotation in progress
	}

	const errorPrefix = "HTTP log rotation: "
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
					errorLog.Printf("%vstatus file error: %v", errorPrefix, err)
				}
			} else {
				now := time.Now()
				sfd := sfi.ModTime()
				if now.Sub(sfd) > logCfg.MaxAgeDuration {
					if logCfg.BackupDays == 0 {
						rotate = true
					} else if now.Year() != sfi.ModTime().Year() && now.Month() != sfi.ModTime().Month() && now.Day() != sfi.ModTime().Day() {
						// prolong till the end of the day
						rotate = true
					}
				}
			}
		}
		if logCfg.MaxSizeBytes > 0 && fi.Size() > logCfg.MaxSizeBytes {
			rotate = true
		}
	}

	if rotate {
		backgroundBlock.Add(1)
		go func() {
			defer backgroundBlock.Done()
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
					errorLog.Printf("%vbackup file error: %v", errorPrefix, err)
				}
				return
			}

			if logCfg.MaxAgeDuration > 0 {
				// touch status file
				// change it once https://github.com/golang/go/issues/32558 will be fixed
				defer func() {
					if err := os.Chtimes(statusFile, now, now); err != nil {
						errorLog.Printf("%vstatus file touch error: %v", errorPrefix, err)
					}
				}()
			}

			currentDate := now.Format("2006-01-02")
			extension := ""
			archive := false
			if logCfg.Archive != "" {
				archive = true
				extension = "." + logCfg.Archive
			}

			// delete old backup files
			var files backupFiles
			if err = files.populate(logFile, extension, errorLog); err != nil {
				errorLog.Printf("%vget backup files error:%v", errorPrefix, err)
			}
			var filesToDelete []string
			var currentOrdinal int
			if logCfg.BackupDays == 0 {
				filesToDelete, currentOrdinal = files.deleteListForBackups(logCfg.Backups, currentDate)
			} else {
				filesToDelete, currentOrdinal = files.deleteListForDaysBackup(logCfg.BackupDays, logCfg.Backups, currentDate)
			}
			for _, file := range filesToDelete {
				if err = os.Remove(file); err != nil {
					errorLog.Printf("%vdelete '%v' file error: %v", errorPrefix, file, err)
				}
			}

			// rename/archive backup file
			historyFile := logFile + "-" + currentDate
			if currentOrdinal > 0 {
				historyFile += "_" + strconv.Itoa(currentOrdinal)
			}
			historyFileName := filepath.Base(historyFile)
			historyFile += extension

			if archive {
				err = zipFilesToFile(historyFile, logCfg.FileMode, []fileToArchive{{name: historyFileName, path: backupFile}})
				if err == nil {
					err = os.Remove(backupFile)
				}
			} else {
				for i := 0; i < 6; i++ {
					err = os.Rename(backupFile, historyFile)
					if err == nil || os.IsNotExist(err) {
						break
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
			if err != nil {
				if !os.IsNotExist(err) {
					errorLog.Printf("%vhistory file '%v' error: %v", errorPrefix, historyFile, err)
				}
				return
			}
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

func (f *backupFiles) populate(logFile, extension string, errorLog *log.Logger) error {
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
				ordinal, err := 0, error(nil)
				if m[3] != "" {
					ordinal, err = strconv.Atoi(m[3])
				}
				if err == nil {
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
		f.files = nil
		return fmt.Errorf("%v", errors)
	}
	sort.Sort(f)
	return nil
}

func (f *backupFiles) enumFilesForDelete(currentDate string, processFile func(file backupFileInfo, files *[]string) bool) (files []string, currentOrdinal int) {
	checkDate := true
	for _, file := range f.files {
		setOrdinal := true
		if checkDate {
			rc := strings.Compare(currentDate, file.date)
			switch {
			case rc < 0:
				files = append(files, file.path)
				continue
			case rc == 0:
				setOrdinal = true
			default:
				checkDate = false
			}
		}
		if processFile(file, &files) {
			if setOrdinal {
				currentOrdinal = file.ordinal + 1
				checkDate = false
			}
		}
	}
	return
}

func (f *backupFiles) deleteListForBackups(backups uint32, currentDate string) (files []string, currentOrdinal int) {
	backups--
	return f.enumFilesForDelete(currentDate, func(file backupFileInfo, files *[]string) (keep bool) {
		if backups > 0 {
			backups--
			keep = true
		} else {
			*files = append(*files, file.path)
		}
		return
	})
}

func (f *backupFiles) deleteListForDaysBackup(backupDays uint32, backupsPerDays uint32, currentDate string) (files []string, currentOrdinal int) {
	if backupsPerDays == 0 {
		backupsPerDays = 1
	}
	prevDate, daysCount, perDayCount := "", uint32(0), uint32(1)
	return f.enumFilesForDelete(currentDate, func(file backupFileInfo, files *[]string) (keep bool) {
		if prevDate == file.date {
			if perDayCount >= backupsPerDays {
				*files = append(*files, file.path)
			} else {
				perDayCount++
				keep = true
			}
		} else {
			if daysCount >= backupDays {
				*files = append(*files, file.path)
			} else {
				perDayCount = 1
				if file.date == currentDate {
					perDayCount = 2
				}
				daysCount++
				prevDate = file.date
				keep = true
			}
		}
		return
	})
}
