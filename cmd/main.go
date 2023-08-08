package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mengdu/gocrontab/injector"
	"github.com/mengdu/mo"
)

var crontab = flag.String("crontab", "", "Crontab file path")

func init() {
	flag.Parse()
	mo.Std.Formater = &mo.TextForamter{
		EnableLevel: true,
		ShortLevel:  true,
		// DisableLevelIcon: true,
		EnableTime: true,
	}
	if *crontab == "" {
		mo.Error("Pleace provide crontab file path")
		os.Exit(1)
	}
	crontabFile, err := filepath.Abs(*crontab)
	if err != nil {
		mo.Errorf("Error: %s", err)
		os.Exit(1)
	}
	if stat, err := os.Stat(crontabFile); os.IsNotExist(err) {
		mo.Errorf("Not found crontab file: %s", crontabFile)
		os.Exit(1)
	} else if stat.IsDir() {
		mo.Errorf("\"%s\" is not a crontab file", crontabFile)
		os.Exit(1)
	}
	mo.Infof("Crontab file: %s", crontabFile)
	*crontab = crontabFile
}

func main() {
	inject, err := injector.Build()
	if err != nil {
		mo.Errorf("Inject error: %s", err)
		os.Exit(1)
	}

	if err := inject.Manager.Start(*crontab); err != nil {
		mo.Errorf("Start error: %s", err)
		os.Exit(1)
	}

	mo.Success("Server start successfully")
	sock := "/tmp/gocrond.sock"
	defer os.Remove(sock)
	go func() {
		if err := inject.Socket.Start(sock, inject.Manager); err != nil {
			mo.Errorf("Socket start error: %s", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	os.Remove(sock)
	mo.Info("Shutdown Server ...")
}
