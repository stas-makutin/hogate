// +build !windows

package main

func setServiceParameter(svcName, name, value string) error {
	return nil
}

func getServiceParameter(svcName, name) (string, error) {
	return "", nil
}

