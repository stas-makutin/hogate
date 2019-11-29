// +build !windows

package main

import "path/filepath"

func defaultConfigFile() string {
	return filepath.Join("/etc", appName, appName+".yml")
}
