package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/google/wire"
	"github.com/mengdu/gocrontab/internal/core"
)

func New() *Server {
	return &Server{}
}

var Set = wire.NewSet(New)
var startAt = time.Now()

type Server struct{}

func (s *Server) Start(address string, cron *core.Manager) error {
	handler := createHandler(cron)
	listener, err := net.Listen("unix", address)
	if err != nil {
		opErr, ok := err.(*net.OpError)
		if ok && opErr.Op == "listen" && opErr.Net == "unix" && errors.Is(opErr.Err, syscall.EADDRINUSE) {
			conn, err2 := net.Dial("unix", address)
			if err2 != nil {
				if errors.Is(err2, syscall.ECONNREFUSED) || errors.Is(err2, syscall.ENOENT) {
					if err := os.Remove(address); err != nil {
						fmt.Println("retry fail:", err)
						return err
					}
					listen, err2 := net.Listen("unix", address)
					if err2 != nil {
						return err
					}
					listener = listen
				} else {
					return err
				}
			} else {
				return err
			}
			if conn != nil {
				conn.Close()
			}
		} else {
			return err
		}
	}
	defer listener.Close()

	srv := http.Server{
		Handler: handler,
	}
	if err := srv.Serve(listener); err != nil {
		return err
	}
	return nil
}
