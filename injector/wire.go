//go:build wireinject
// +build wireinject

package injector

import (
	"github.com/google/wire"
	"github.com/mengdu/gocrontab/internal/core"
	"github.com/mengdu/gocrontab/internal/socket"
)

var set = wire.NewSet(
	wire.Struct(new(Injector), "*"),
	core.Set,
	socket.Set,
)

type Injector struct {
	Manager *core.Manager
	Socket  *socket.Server
}

func Build() (*Injector, error) {
	wire.Build(set)
	return new(Injector), nil
}
