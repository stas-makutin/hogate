// +build !windows

package main

import "path/filepath"

const NewLine = "\n"

func defaultConfigFile() string {
	return filepath.Join("/etc", appName, appName+".yml")
}
