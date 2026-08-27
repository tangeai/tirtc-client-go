package runtimelease

import (
	"errors"
	"sync"
)

var ErrConflict = errors.New("runtime core configuration conflicts with an active product")

type Product uint8

const (
	RTC Product = iota
	CloudStorage
)

type Configuration struct {
	AppID             string
	Endpoint          string
	CacheDir          string
	ConsoleLogEnabled bool
}

var leases struct {
	sync.Mutex
	products [2]*Configuration
}

// Init serializes the two public product initializers and commits a lease only
// after the native initializer succeeds. CacheDir and ConsoleLogEnabled are the
// shared Runtime Core configuration; AppID and Endpoint remain product-local.
func Init(product Product, configuration Configuration, start func() error) error {
	leases.Lock()
	defer leases.Unlock()
	current := leases.products[product]
	if current != nil {
		if *current == configuration {
			return nil
		}
		return ErrConflict
	}
	for _, active := range leases.products {
		if active != nil && (active.CacheDir != configuration.CacheDir ||
			active.ConsoleLogEnabled != configuration.ConsoleLogEnabled) {
			return ErrConflict
		}
	}
	if err := start(); err != nil {
		return err
	}
	committed := configuration
	leases.products[product] = &committed
	return nil
}

func Shutdown(product Product, stop func() error) error {
	leases.Lock()
	defer leases.Unlock()
	if err := stop(); err != nil {
		return err
	}
	leases.products[product] = nil
	return nil
}
