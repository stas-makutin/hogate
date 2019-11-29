package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kardianos/service"
)

const appName = "hogate"

func usage() {
	fmt.Printf(
		`Home HTTP Gateway service.

Usage: [Action] [Options]

Actions:

install [option]
  Install the service
uninstall
  Uninstall the service
start
  Start the service
stop
  Stop the service
run [option]
  Execut as console application

Options:
-h, --help
  Print this message
-c, --config <file name>
  Path to configuration yaml file. Works only for install and run actions.
  Default: %v
`,
		defaultConfigFile(),
	)
}

type application struct {
	configFile string
	logger     service.Logger
	stop       func()
}

func (app *application) run() {
	if app.configFile == "" {
		app.configFile = defaultConfigFile()
	}
	app.logger.Info(app.configFile)
	app.stop()
}

func (app *application) Start(s service.Service) error {
	if app.configFile == "" {
		argc := len(os.Args)
		for i := 1; i < argc; i++ {
			arg := os.Args[i]
			switch arg {
			case "-c", "--config":
				if i++; i < argc {
					app.configFile = os.Args[i]
				}
				break
			}
		}
	}
	app.stop = func() {
		s.Stop()
	}
	go app.run()
	return nil
}

func (app *application) Stop(s service.Service) error {
	return nil
}

func main() {
	app := &application{}
	action := ""

	argc := len(os.Args)
	for i := 1; i < argc; i++ {
		arg := os.Args[i]
		switch arg {
		case "-h", "--help":
			if service.Interactive() {
				usage()
				os.Exit(100)
			}
		case "-c", "--config":
			if i++; i < argc {
				app.configFile = os.Args[i]
			}
		default:
			if action == "" {
				action = arg
			}
		}
	}

	var arguments []string
	if app.configFile != "" {
		arguments = []string{"--config", app.configFile}
	}

	svcConfig := &service.Config{
		Name:        "hogate",
		DisplayName: "Home HTTP Gateway",
		Description: "Home HTTP Gateway.",
		Arguments:   arguments,
	}

	svc, err := service.New(app, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	app.logger, err = svc.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	if service.Interactive() {
		if action == "run" {
			app.run()
		} else {
			err = service.Control(svc, action)
			if err != nil {
				fmt.Println(err)
			}
		}
		return
	}
	err = svc.Run()
	if err != nil {
		app.logger.Error(err)
	}
}
