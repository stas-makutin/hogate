package main

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var zwCmd = ""

var zwCommandTimeout = 2500 // milliseconds

const (
	zwSuccess = iota
	zwQueryFailed
	zwPortFailed
	zwNoResources
	zwNoParameter
	zwBusy
	zwSystemError
)

var zwCommandLock sync.Mutex

func validateZwCmdConfig(cfgError configError) {
	if config.ZwCmd == nil {
		return
	}

	if config.ZwCmd.Path != "" {
		if _, err := os.Stat(config.ZwCmd.Path); err != nil {
			cfgError(fmt.Sprintf("zwCmd.path '%v' is not exists/accessible.", config.ZwCmd.Path))
		} else {
			zwCmd = config.ZwCmd.Path
		}
	}

	if config.ZwCmd.Timeout != 0 {
		if config.ZwCmd.Timeout < 0 {
			cfgError(fmt.Sprintf("zwCmd.timeout '%v' could not be negative.", config.ZwCmd.Timeout))
		} else {
			zwCommandTimeout = config.ZwCmd.Timeout
		}
	}
}

func zwCommand(arg ...string) (retCode int, attributes map[string]string) {
	retCode = zwSystemError
	attributes = make(map[string]string)

	if zwCmd == "" {
		return
	}

	zwCommandLock.Lock()
	defer zwCommandLock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(zwCommandTimeout))
	defer cancel()

	output, _ := exec.CommandContext(ctx, zwCmd, append([]string{"--timeout", strconv.Itoa(zwCommandTimeout), "--xml"}, arg...)...).Output()
	if output != nil {
		r := bytes.NewReader(output)
		d := xml.NewDecoder(r)
		for {
			t, err := d.Token()
			if err != nil {
				if err != io.EOF {
					retCode = zwSystemError
				}
				break
			}
			switch t := t.(type) {
			case xml.StartElement:
				if t.Name.Local == "zwt" {
					for _, v := range t.Attr {
						attributes[v.Name.Local] = v.Value
						switch v.Name.Local {
						case "success":
							if v.Value == "1" {
								retCode = zwSuccess
							}
						case "code":
							switch v.Value {
							case "0":
								retCode = zwSuccess
							case "2147483643":
								retCode = zwQueryFailed
							case "2147483644":
								retCode = zwPortFailed
							case "2147483645":
								retCode = zwNoResources
							case "2147483646":
								retCode = zwNoParameter
							default:
								retCode = zwQueryFailed
							}
						}
					}
				}
			}
		}
	}

	return
}

func zwBasicSet(nodeId byte, level byte) int {
	code, _ := zwCommand("basic", strconv.Itoa(int(nodeId)), strconv.Itoa(int(level)))
	return code
}

func zwBasicGet(nodeId byte) (int, byte) {
	code, attr := zwCommand("basic", strconv.Itoa(int(nodeId)), "--get")
	if code == zwSuccess {
		if value, ok := attr["value"]; ok {
			if v, err := strconv.Atoi(value); err == nil {
				return code, byte(v)
			}
		}
	}
	return code, 0
}
