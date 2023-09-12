//go:build wireinject
// +build wireinject

package injector

import (
	"github.com/google/wire"
	"github.com/mengdu/gocrontab/internal/core"
	"github.com/mengdu/gocrontab/internal/server"
)

var set = wire.NewSet(
	wire.Struct(new(Injector), "*"),
	core.Set,
	server.Set,
)

type Injector struct {
	Manager *core.Manager
	Server  *server.Server
}

func Build() (*Injector, error) {
	wire.Build(set)
	return new(Injector), nil
}
