package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
)

const appName = "hogate"

const configFileParameter = "configFile"

func usage() {
	fmt.Printf(
		`Home HTTP Gateway service.

Usage: [Options]

Options:
-h, --help
  Print this message.
-c, --config <file path>
  Path to configuration yaml file. 
  Default: %v
-s, --service <action>
  Control the service. Action could be any of:
  install, uninstall, start, stop, restart, run
`,
		defaultConfigFile(),
	)
}

func defaultConfigFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(homeDir, appName, appName+".yml")
}

type application struct {
	configFile string
	logger     service.Logger
}

func (app *application) run() {
	if app.configFile == "" {
		app.configFile, _ = getServiceParameter(appName, configFileParameter)
		if app.configFile == "" {
			app.configFile = defaultConfigFile()
		}
	}
	app.logger.Info(app.configFile)
}

func (app *application) Start(s service.Service) error {
	go app.run()
	return nil
}

func (app *application) Stop(s service.Service) error {
	return nil
}

func main() {

	svcConfig := &service.Config{
		Name:        "hogate",
		DisplayName: "Home HTTP Gateway",
		Description: "Home HTTP Gateway.",
	}

	app := &application{}
	svc, err := service.New(app, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	app.logger, err = svc.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	if service.Interactive() {
		var action string

		argc := len(os.Args)
		for i := 0; i < argc; i++ {
			arg := os.Args[i]
			switch arg {
			case "-h", "--help":
				usage()
				os.Exit(100)
			case "-c", "--config":
				if i++; i < argc {
					app.configFile = os.Args[i]
				}
			case "-s", "--service":
				if i++; i < argc {
					action = os.Args[i]
				}
			}
		}

		saveParameters := true
		switch action {
		case "install":
			err = svc.Install()
		case "uninstall":
			saveParameters = false
			err = svc.Uninstall()
		case "start":
			err = svc.Start()
		case "stop":
			err = svc.Stop()
		case "restart":
			err = svc.Restart()
		case "run":
			saveParameters = false
			app.run()
			return
		}

		if err != nil {
			log.Println(err)
		}

		if saveParameters && app.configFile != "" {
			err := setServiceParameter(svcConfig.Name, configFileParameter, app.configFile)
			if err != nil {
				fmt.Printf("Path to configuration file wasn't updated: %v", err)
			}
		}

		return
	}
	err = svc.Run()
	if err != nil {
		app.logger.Error(err)
	}
}
