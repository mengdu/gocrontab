// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package injector

import (
	"github.com/google/wire"
	"github.com/mengdu/gocrontab/internal/core"
	"github.com/mengdu/gocrontab/internal/socket"
)

// Injectors from wire.go:

func Build() (*Injector, error) {
	manager := core.New()
	server := socket.New()
	injector := &Injector{
		Manager: manager,
		Socket:  server,
	}
	return injector, nil
}

// wire.go:

var set = wire.NewSet(wire.Struct(new(Injector), "*"), core.Set, socket.Set)

type Injector struct {
	Manager *core.Manager
	Socket  *socket.Server
}
