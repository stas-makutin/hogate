package main

import (
	"github.com/kardianos/service"
)

type program struct{}

func (p *program) run() {
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "hogate",
		DisplayName: "Home HTTP Gateway",
		Description: "Home HTTP Gateway",
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		//log.Fatal(err)
	}
	logger, err := s.Logger(nil)
	if err != nil {
		//log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
