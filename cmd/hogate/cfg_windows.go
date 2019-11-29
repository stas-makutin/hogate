// +build windows

package main

import (
	"os"
	"path/filepath"
)

func defaultConfigFile() string {
	return filepath.Join(filepath.Dir(os.Args[0]), appName+".yml")
}
