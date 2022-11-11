package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type suffixMultiplier struct {
	suffix     string
	multiplier float64
}

var sizeSuffixes []suffixMultiplier = []suffixMultiplier{
	{"kib", 1024}, {"kb", 1024}, {"ki", 1024}, {"k", 1024},
	{"mib", 1024 * 1024}, {"mb", 1024 * 1024}, {"mi", 1024 * 1024}, {"m", 1024 * 1024},
	{"gib", 1024 * 1024 * 1024}, {"gb", 1024 * 1024 * 1024}, {"gi", 1024 * 1024 * 1024}, {"g", 1024 * 1024 * 1024},
	{"tib", 1024 * 1024 * 1024 * 1024}, {"tb", 1024 * 1024 * 1024 * 1024}, {"ti", 1024 * 1024 * 1024 * 1024}, {"t", 1024 * 1024 * 1024 * 1024},
	{"pib", 1024 * 1024 * 1024 * 1024 * 1024}, {"pb", 1024 * 1024 * 1024 * 1024 * 1024}, {"pi", 1024 * 1024 * 1024 * 1024 * 1024}, {"p", 1024 * 1024 * 1024 * 1024 * 1024},
}

var timeSuffixes []suffixMultiplier = []suffixMultiplier{
	{"microseconds", float64(time.Microsecond)}, {"microsecond", float64(time.Microsecond)},
	{"milliseconds", float64(time.Millisecond)}, {"millisecond", float64(time.Millisecond)},
	{"minutes", float64(time.Minute)}, {"minute", float64(time.Minute)},
	{"hours", float64(time.Hour)}, {"hour", float64(time.Hour)},
	{"days", float64(24 * time.Hour)}, {"day", float64(24 * time.Hour)},
	{"seconds", float64(time.Second)}, {"second", float64(time.Second)},
	{"mks", float64(time.Microsecond)}, {"ms", float64(time.Millisecond)},
	{"m", float64(time.Minute)}, {"h", float64(time.Hour)}, {"d", float64(24 * time.Hour)}, {"s", float64(time.Second)},
}

func parseSuffixed(value string, suffixes []suffixMultiplier) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	value = strings.ToLower(value)

	var multiplier float64 = 1
	for _, v := range suffixes {
		if strings.HasSuffix(value, v.suffix) {
			value = strings.TrimSpace(value[0 : len(value)-len(v.suffix)])
			multiplier = v.multiplier
			break
		}
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	return int64(v * multiplier), nil
}

func parseSizeString(size string) (int64, error) {
	return parseSuffixed(size, sizeSuffixes)
}

func parseTimeDuration(duration string) (time.Duration, error) {
	v, err := parseSuffixed(duration, timeSuffixes)
	return time.Duration(v), err
}

type fileToArchive struct {
	name, path string
}

func zipFilesToWriter(w *zip.Writer, files []fileToArchive) error {
	for _, file := range files {
		err := func() error {
			src, err := os.Open(file.path)
			if err != nil {
				return err
			}
			defer src.Close()

			dest, err := w.Create(file.name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(dest, src); err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func zipFilesToFile(zipFile string, perm os.FileMode, files []fileToArchive) error {
	f, err := os.OpenFile(zipFile, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	err = func() error {
		defer f.Close()
		zw := zip.NewWriter(f)
		err := zipFilesToWriter(zw, files)
		errClose := zw.Close()
		if err != nil {
			return err
		}
		return errClose
	}()
	if err != nil {
		os.Remove(zipFile)
	}
	return err
}

func randomString(size int, alphabet []rune) string {
	if size <= 0 {
		return ""
	}
	b := make([]rune, size)
	l := int64(len(alphabet))
	for i := range b {
		b[i] = alphabet[rand.Int63n(l)]
	}
	return string(b)
}

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	s := string(b)
	return s[1 : len(s)-1]
}

func parseJSONRequest(v interface{}, w http.ResponseWriter, r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusForbidden)
		return false
	}
	d := json.NewDecoder(r.Body)
	if err := d.Decode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return false
	}
	return true
}

func parseOptions(options string, accept func(option string) bool) bool {
	for _, option := range strings.FieldsFunc(options, func(r rune) bool { return r == ',' || r == ';' || unicode.IsSpace(r) }) {
		if option != "" {
			if !accept(option) {
				return false
			}
		}
	}
	return true
}
