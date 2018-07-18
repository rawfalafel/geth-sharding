package shared

import (
	"context"
	"fmt"
	"reflect"
	log "github.com/sirupsen/logrus"
)

// ServiceRegistry provides a useful pattern for managing services.
// It allows for ease of dependency management and ensures services
// dependent on others use the same references in memory.
type ServiceRegistry struct {
	services     map[reflect.Type]Service // map of types to services.
	serviceTypes []reflect.Type           // keep an ordered slice of registered service types.
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewServiceRegistry starts a registry instance for convenience
func NewServiceRegistry() *ServiceRegistry {
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceRegistry{
		services: make(map[reflect.Type]Service),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// GetContext returns the root context for all services.
// Contexts within services must be derived from this context.
func (s *ServiceRegistry) GetContext() context.Context {
	return s.ctx
}

// StartAll initialized each service in order of registration.
func (s *ServiceRegistry) StartAll() {
	for _, kind := range s.serviceTypes {
		s.services[kind].Start()
	}
}

// StopAll ends every service, logging a panic if any of them fail to stop.
func (s *ServiceRegistry) StopAll() {
	log.Info("StopAll")
	s.cancel()
}

// RegisterService appends a service constructor function to the service
// registry.
func (s *ServiceRegistry) RegisterService(service Service) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}

// FetchService takes in a struct pointer and sets the value of that pointer
// to a service currently stored in the service registry. This ensures the input argument is
// set to the right pointer that refers to the originally registered service.
func (s *ServiceRegistry) FetchService(service interface{}) error {
	if reflect.TypeOf(service).Kind() != reflect.Ptr {
		return fmt.Errorf("input must be of pointer type, received value type instead: %T", service)
	}
	element := reflect.ValueOf(service).Elem()
	if running, ok := s.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service: %T", service)
}
