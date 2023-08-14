package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mengdu/gocrontab/internal/core"
	"github.com/mengdu/mo"
	"github.com/olekukonko/tablewriter"
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
			args := bytes.SplitN(message, []byte("=>:"), 2)
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
	if _, err := e.writer.Write([]byte(msgType + "=>:")); err != nil {
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
	sock := flag.String("sock", "/tmp/gocrond.sock", "Socket file")
	subCommand := flag.NewFlagSet("sub", flag.ExitOnError)
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
	}
	e := Client{}
	e.On("ping", func(msg []byte) {
		mo.Debug(string(msg))
		os.Exit(0)
	})

	e.On("ls", func(msg []byte) {
		list := []core.Job{}
		if err := json.Unmarshal(msg, &list); err != nil {
			mo.Panicf("Error: %s", err)
		}
		// for _, item := range list {
		// 	mo.Infof("%t %s %s %s %s", item.Running, item.Spec, item.ID, item.Cmd, item.Title)
		// }
		mo.Infof("Total %d jobs", len(list))
		table := tablewriter.NewWriter(os.Stdout)
		table.SetRowLine(true)
		table.SetHeader([]string{"ID", "Spec", "Running", "Title"})
		for _, item := range list {
			table.Append([]string{item.ID, item.Spec, fmt.Sprintf("%t", item.Running), item.Title})
		}
		table.Render()
		os.Exit(0)
	})

	e.On("exec", func(msg []byte) {
		if string(msg) != "ok" {
			mo.Errorf("%s", msg)
			os.Exit(1)
		}
		mo.Successf("Executing ok")
		os.Exit(0)
	})

	conn, err := e.Dial(*sock)
	if err != nil {
		mo.Errorf("Error: %s", err)
		os.Exit(1)
	}
	defer conn.Close()

	switch args[0] {
	case "ping":
		e.Send("ping", nil)
	case "ls":
		e.Send("ls", nil)
	case "exec":
		subCommand.Parse(args[1:])
		subArgs := subCommand.Args()
		if len(subArgs) == 0 {
			mo.Panic("Must provide id")
		}
		e.Send("exec", subArgs[0])
	default:
		mo.Panicf("Unknown command: %s", args[0])
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	mo.Info("Shutdown Server ...")
}
