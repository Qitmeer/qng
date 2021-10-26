package vm

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/vm/chainvm"
	"github.com/Qitmeer/qng/vm/common"
	"io/ioutil"
	"path/filepath"
)

type Factory interface {
	New(context.Context) (interface{}, error)
}

type Service struct {
	service.Service

	events *event.Feed

	factories map[string]Factory

	versions map[string]string

	cfg *config.Config
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
	f, ok := s.factories[id]
	if !ok {
		return nil, fmt.Errorf("No factory:%s", id)
	}
	return f, nil
}

func (s *Service) HasFactory(id string) bool {
	f, err := s.GetFactory(id)
	return err == nil && f != nil
}

func (s *Service) RegisterFactory(vmID string, factory Factory) error {
	if s.HasFactory(vmID) {
		return fmt.Errorf(fmt.Sprintf("Already exists:%s", vmID))
	}

	s.factories[vmID] = factory

	log.Debug(fmt.Sprintf("Adding factory for vm %s", vmID))

	vm, err := factory.New(nil)
	if err != nil {
		return err
	}

	commonVM, ok := vm.(common.VM)
	if !ok {
		return nil
	}

	version, err := commonVM.Version()
	if err != nil {
		log.Error(fmt.Sprintf("fetching version for %q errored with: %s", vmID, err))

		if err := commonVM.Shutdown(); err != nil {
			return fmt.Errorf("shutting down VM errored with: %s", err)
		}
		return nil
	}
	s.versions[vmID] = version

	return nil
}

func (s *Service) Versions() (map[string]string, error) {
	return s.versions, nil
}

func (s *Service) registerVMs() error {
	if len(s.cfg.PluginDir) <= 0 {
		return nil
	}
	files, err := ioutil.ReadDir(s.cfg.PluginDir)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("Register Virtual Machines from:%s num:%d", s.cfg.PluginDir, len(files)))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		// Strip any extension from the file. This is to support windows .exe
		// files.
		name = name[:len(name)-len(filepath.Ext(name))]
		// Skip hidden files.
		if len(name) == 0 {
			continue
		}

		if s.HasFactory(name) {
			continue
		}

		if err = s.RegisterFactory(name, &chainvm.Factory{
			Path: filepath.Join(s.cfg.PluginDir, file.Name()),
		}); err != nil {
			return err
		}
	}

	return nil
}

func NewService(cfg *config.Config, events *event.Feed) (*Service, error) {
	ser := Service{
		events:    events,
		factories: make(map[string]Factory),
		versions:  make(map[string]string),
		cfg:       cfg,
	}

	if err := ser.registerVMs(); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return &ser, nil
}
