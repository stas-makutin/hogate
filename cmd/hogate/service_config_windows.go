package main

import (
	"golang.org/x/sys/windows/registry"
)

func setServiceParameter(svcName, name, value string) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\`+svcName, registry.WRITE)
	if err != nil {
		return err
	}
	defer key.Close()
	subKey, _, err := registry.CreateKey(key, "Parameters", registry.WRITE)
	if err != nil {
		return err
	}
	defer subKey.Close()
	return subKey.SetStringValue(name, value)
}

func getServiceParameter(svcName, name string) (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\`+svcName+`\Parameters`, registry.READ)
	if err != nil {
		return "", err
	}
	v, _, err := key.GetStringValue(name)
	return v, err
}
