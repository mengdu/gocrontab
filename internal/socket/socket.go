package socket

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
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

type Conn struct {
	net.Conn
	Data []byte
}

type EventHandler = func(c Conn) ([]byte, error)

type Socket struct {
	handlers map[string]EventHandler
}

func (e *Socket) On(msgType string, handler EventHandler) {
	if e.handlers == nil {
		e.handlers = make(map[string]EventHandler)
	}
	e.handlers[msgType] = handler
}

func (e *Socket) Listen(address string) error {
	srv, err := net.Listen("unix", address)
	if err != nil {
		return err
	}
	defer srv.Close()
	defer os.Remove(address)

	for {
		conn, err := srv.Accept()
		if err != nil {
			mo.Errorf("Error: %s", err)
			continue
		}
		go func() {
			defer conn.Close()
			reader := bufio.NewReader(conn)
			writer := bufio.NewWriter(conn)
			for {
				message, err := reader.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						mo.Debugf("Connection closed by client")
						return
					}
					mo.Errorf("Failed to read message: %s", err)
					return
				}
				message = bytes.TrimRight(message, "\n")
				args := bytes.SplitN(message, []byte("=>:"), 2)
				if len(args) != 2 {
					mo.Errorf("Invalid message: %s", message)
					return
				}
				cmd := string(args[0])
				mo.Debugf("Received: %s => %s", cmd, args[1])
				handler, ok := e.handlers[cmd]
				if !ok {
					mo.Errorf("Unknown command: %s", cmd)
					return
				}
				result, err := handler(Conn{
					Conn: conn,
					Data: args[1],
				})
				if err != nil {
					mo.Errorf("Handler Error: %s", err)
					return
				}
				if len(result) != 0 {
					writer.Write([]byte(fmt.Sprintf("%s=>:", cmd)))
					if _, err := writer.Write(result); err != nil {
						mo.Errorf("Write Error: %s", err)
						return
					}
					if err := writer.WriteByte('\n'); err != nil {
						mo.Errorf("WriteByte Error: %s", err)
						return
					}
					if err := writer.Flush(); err != nil {
						mo.Errorf("Flush Error: %s", err)
						return
					}
				}
			}
		}()
	}
}

func (s *Server) Start(address string, cron *core.Manager) error {
	srv := &Socket{}
	srv.On("ping", func(c Conn) ([]byte, error) {
		return []byte("pong"), nil
	})

	srv.On("ls", func(c Conn) ([]byte, error) {
		return json.Marshal(cron.GetJobs())
	})

	srv.On("exec", func(c Conn) ([]byte, error) {
		err := cron.Exec(strings.Trim(string(c.Data), "\""))
		if err != nil {
			return []byte(fmt.Sprintf("%s", err)), nil
		}
		return []byte("ok"), nil
	})

	err := srv.Listen(address)
	if err != nil {
		return err
	}
	return nil
}
