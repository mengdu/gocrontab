//go:build wireinject
// +build wireinject

package injector

import (
	"github.com/google/wire"
	"github.com/mengdu/gocrontab/internal/core"
)

var set = wire.NewSet(
	wire.Struct(new(Injector), "*"),
	core.Set,
)

type Injector struct {
	Manager *core.Manager
}

func Build() (*Injector, error) {
	wire.Build(set)
	return new(Injector), nil
}
