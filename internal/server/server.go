package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/google/wire"
	"github.com/mengdu/gocrontab/internal/core"
	"github.com/mengdu/mo"
)

func New() *Server {
	return &Server{}
}

var Set = wire.NewSet(New)

type Server struct{}

func (s *Server) Start(address string, cron *core.Manager) error {
	handler := &MyHandler{}
	handler.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("Pong"))
	})

	handler.Get("/ls", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		buf, err := json.Marshal(map[string]interface{}{
			"ret":  0,
			"msg":  "ok",
			"list": cron.GetJobs(),
		})
		if err != nil {
			mo.Error(err)
			buf, _ := json.Marshal(map[string]interface{}{
				"ret": -1,
				"msg": err.Error(),
			})
			w.Write(buf)
			return
		}
		w.Write(buf)
	})

	handler.Post("/exec", func(w http.ResponseWriter, req *http.Request) {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			mo.Error(err)
		}
		cid := strings.Trim(string(data), "\"")
		if cid == "" {
			buf, _ := json.Marshal(map[string]interface{}{
				"ret": 1,
				"msg": "params error",
			})
			w.Write(buf)
			return
		}
		err = cron.Exec(cid)
		if err != nil {
			buf, _ := json.Marshal(map[string]interface{}{
				"ret": -1,
				"msg": err.Error(),
			})
			w.Write(buf)
			return
		}
		buf, _ := json.Marshal(map[string]interface{}{
			"ret": 0,
			"msg": "ok",
		})
		w.Write(buf)
	})

	listener, err := net.Listen("unix", address)
	if err != nil {
		return err
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

type HFunc = func(w http.ResponseWriter, req *http.Request)
type MyHandler struct {
	routes map[string]HFunc
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	mo.Debugf("%s %s", req.Method, req.URL.String())
	fn, ok := h.routes[fmt.Sprintf("%s %s", req.Method, req.URL.Path)]
	if ok {
		fn(w, req)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 Not Found\n"))
}

func (h *MyHandler) Get(path string, fn HFunc) {
	if h.routes == nil {
		h.routes = make(map[string]HFunc)
	}
	h.routes[fmt.Sprintf("GET %s", path)] = fn
}

func (h *MyHandler) Post(path string, fn HFunc) {
	if h.routes == nil {
		h.routes = make(map[string]HFunc)
	}
	h.routes[fmt.Sprintf("POST %s", path)] = fn
}
