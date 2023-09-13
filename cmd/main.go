package main

import (
	"flag"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/mengdu/gocrontab/injector"
	"github.com/mengdu/mo"
)

var crontab = flag.String("c", "", "Crontab file path")
var sockFile = flag.String("sock", "", "Socket file")
var debug = flag.Bool("debug", false, "Debug mode")

func init() {
	flag.Parse()
	// init reset mo
	mo.Std.Formater = &mo.TextForamter{
		EnableLevel: true,
		ShortLevel:  true,
		// DisableLevelIcon: true,
		EnableTime: true,
	}
	if *debug {
		mo.Std.Level = mo.LEVEL_DEBUG
	} else {
		mo.Std.Level = mo.LEVEL_SUCCESS
	}
	if *crontab == "" {
		mo.Panicf("Please specify a task configuration file through the '-c' parameter")
	}
	crontabFile, err := filepath.Abs(*crontab)
	if err != nil {
		mo.Panicf("Error: %s", err)
	}
	if stat, err := os.Stat(crontabFile); os.IsNotExist(err) {
		mo.Panicf("Not found crontab file: %s", crontabFile)
	} else if stat.IsDir() {
		mo.Panicf("\"%s\" is not a crontab file", crontabFile)
	}
	mo.Infof("Crontab file: %s", crontabFile)
	*crontab = crontabFile
}

func main() {
	inject, err := injector.Build()
	if err != nil {
		mo.Panicf("Inject error: %s", err)
	}

	if err := inject.Manager.Start(*crontab); err != nil {
		mo.Panicf("Start error: %s", err)
	}

	sock := ""
	if *sockFile == "" {
		usr, err := user.Current()
		if err != nil {
			mo.Panic(err)
		}
		sock = filepath.Join(usr.HomeDir, "gocrond.sock")
	} else {
		sock = *sockFile
	}

	defer os.Remove(sock)
	go func() {
		if err := inject.Server.Start(sock, inject.Manager); err != nil {
			mo.Panicf("Socket start error: %s", err)
		}
	}()
	mo.Success("Server start successfully")
	mo.Infof("Process pid: %d, ppid: %d", os.Getpid(), os.Getppid())
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	os.Remove(sock)
	mo.Info("Shutdown Server ...")
}
