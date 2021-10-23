package vm

import (
	"context"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/node/service"
)

type Factory interface {
	New(*context.Context) (interface{}, error)
}

type Service struct {
	service.Service

	events *event.Feed

	factories map[string]Factory

	versions map[string]string
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

func (s *Service) GetFactory(id string) (Factory, error) {

}

func (s *Service) RegisterFactory(id string, factory Factory) error {

}

func (s *Service) Versions() (map[string]string, error) {

}

func NewService(cfg *config.Config, events *event.Feed) (*Service, error) {
	ser := Service{
		events:    events,
		factories: make(map[string]Factory),
		versions:  make(map[string]string),
	}

	return &ser, nil
}
