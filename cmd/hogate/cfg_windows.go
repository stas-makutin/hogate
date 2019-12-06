// +build windows

package main

import (
	"os"
	"path/filepath"
)

const NewLine = "\r\n"

func defaultConfigFile() string {
	return filepath.Join(filepath.Dir(os.Args[0]), appName+".yml")
}
