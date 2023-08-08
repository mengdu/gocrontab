package socket

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"

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
	Writer *bufio.Writer
	Data   []byte
}

func (c *Conn) Read(v any) error {
	return json.Unmarshal(c.Data, v)
}

func (c *Conn) Write(msgType string, v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := c.Writer.Write([]byte(msgType + ":")); err != nil {
		return err
	}
	if _, err := c.Writer.Write(buf); err != nil {
		return err
	}
	if err := c.Writer.WriteByte('\n'); err != nil {
		return err
	}
	if err := c.Writer.Flush(); err != nil {
		return err
	}
	return nil
}

type EventHandler = func(c *Conn)

type Event struct {
	handlers map[string][]EventHandler
}

func (e *Event) On(msgType string, handler EventHandler) {
	if e.handlers == nil {
		e.handlers = make(map[string][]EventHandler)
	}
	e.handlers[msgType] = append(e.handlers[msgType], handler)
}

func (e *Event) Emit(msgType string, c *Conn) {
	for _, handler := range e.handlers[msgType] {
		handler(c)
	}
}

type Socket struct {
	Event
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
				args := bytes.SplitN(message, []byte(":"), 2)
				e.Emit(string(args[0]), &Conn{
					Writer: writer,
					Data:   args[1],
				})
			}
		}()
	}
}

func (s *Server) Start(address string, cron *core.Manager) error {
	srv := &Socket{}
	srv.On("ping", func(c *Conn) {
		mo.Debugf(string(c.Data))
		// jobs := cron.GetJobs()
		// for i := 0; i < len(jobs); i++ {
		// 	mo.Infof("Job: %#v", jobs[i])
		// }
		c.Write("pong", "hi!")
	})
	srv.On("ls", func(c *Conn) {
		c.Write("jobs", cron.GetJobs())
	})
	srv.On("exec", func(c *Conn) {
		id := ""
		if err := c.Read(&id); err != nil {
			mo.Errorf("Error: %s", err)
			return
		}
		cron.Exec(id)
	})

	err := srv.Listen(address)
	if err != nil {
		return err
	}
	return nil
}
