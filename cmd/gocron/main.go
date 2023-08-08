package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mengdu/mo"
)

type EventHandler = func(msg []byte)

type Event struct {
	handlers map[string][]EventHandler
}

func (e *Event) On(msgType string, handler EventHandler) {
	if e.handlers == nil {
		e.handlers = make(map[string][]EventHandler)
	}
	e.handlers[msgType] = append(e.handlers[msgType], handler)
}

func (e *Event) Emit(msgType string, msg []byte) {
	for _, handler := range e.handlers[msgType] {
		handler(msg)
	}
}

type Client struct {
	Event
	writer *bufio.Writer
}

func (e *Client) Dial(address string) (net.Conn, error) {
	conn, err := net.Dial("unix", address)
	if err != nil {
		return conn, err
	}

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	e.writer = writer

	go func() {
		for {
			message, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					e.Emit("error", []byte("Connection closed by client"))
					return
				}
				e.Emit("error", []byte("Failed to read message: "+err.Error()))
				return
			}
			message = bytes.TrimRight(message, "\n")
			args := bytes.SplitN(message, []byte(":"), 2)
			e.Emit(string(args[0]), args[1])
		}
	}()
	return conn, nil
}

func (e *Client) Send(msgType string, v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := e.writer.Write([]byte(msgType + ":")); err != nil {
		return err
	}
	if _, err := e.writer.Write(buf); err != nil {
		return err
	}
	if err := e.writer.WriteByte('\n'); err != nil {
		return err
	}
	if err := e.writer.Flush(); err != nil {
		return err
	}
	return nil
}

func main() {
	e := Client{}
	e.On("pong", func(msg []byte) {
		mo.Debug(string(msg))
		e.Send("exec", "dd4fd0aadcbf1de1ed205e756901a29a")
		// os.Exit(0)
	})
	e.On("jobs", func(msg []byte) {
		mo.Debug(string(msg))
	})

	sock := "/tmp/gocrond.sock"
	conn, err := e.Dial(sock)
	if err != nil {
		mo.Errorf("Error: %s", err)
		os.Exit(1)
	}
	e.Send("ping", "Hello")
	e.Send("ls", "Hello")
	defer conn.Close()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	mo.Info("Shutdown Server ...")
}
