package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mengdu/gocrontab/injector"
	"github.com/mengdu/mo"
)

func main() {
	inject, err := injector.Build()
	if err != nil {
		panic(err)
	}
	inject.Manager.Start()
	mo.Logf("Server start at: %s", time.Now().Format(time.RFC3339))
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	mo.Info("Shutdown Server ...")
}
