package vm

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/node/service"
)

type Service struct {
	service.Service
}

func (s *Service) Start() error {
	log.Info("Starting Virtual Machines Service")
	if err := s.Service.Start(); err != nil {
		return err
	}

	return nil
}

func (s *Service) Stop() error {
	log.Info("Stopping Virtual Machines Service")
	if err := s.Service.Stop(); err != nil {
		return err
	}
	return nil
}

func NewService(cfg *config.Config, events *event.Feed) (*Service, error) {
	ser := Service{}

	return &ser, nil
}
