package core

import (
	"github.com/google/wire"
	"github.com/mengdu/mo"
	"github.com/robfig/cron/v3"
)

func New() *Manager {
	return &Manager{
		// cron: cron.New(cron.WithLocation(time.Local)),
		cron: cron.New(cron.WithSeconds()),
	}
}

var Set = wire.NewSet(New)

type Job struct {
	ID    int64
	Title string
	Spec  string
	Cmd   string
}

type Manager struct {
	cron *cron.Cron
	jobs []Job
}

func (m *Manager) Start() {
	m.cron.AddFunc("* * * * * *", func() {
		mo.Debugf("start %d\n", 1)
	})
	m.cron.Start()
}
