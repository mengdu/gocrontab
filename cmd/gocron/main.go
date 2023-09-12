package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/mengdu/gocrontab/internal/core"
	"github.com/mengdu/mo"
)

func color(s string, start string, end string) string {
	s = strings.ReplaceAll(s, "\u001b[39m", fmt.Sprintf("\u001b[39m\u001b[%sm", start))
	return fmt.Sprintf("\u001b[%sm%s\u001b[%sm", start, s, end)
}

func strHashCode(str string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(str))
	min := 1
	max := 231
	size := max - min + 1
	hashValue := h.Sum32()
	return hashValue%uint32(size) + uint32(min)
}

type HClient struct {
	Client http.Client
}

func (c *HClient) Get(path string, query url.Values) (res *http.Response, body []byte, err error) {
	url := fmt.Sprintf("http://localhost%s", path)
	if len(query) > 0 {
		querystr := query.Encode()
		if querystr != "" {
			url = fmt.Sprintf("%s?%s", url, querystr)
		}
	}
	res, err = c.Client.Get(url)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	return
}

func (c *HClient) Post(path string, query url.Values, payload interface{}) (res *http.Response, body []byte, err error) {
	url := fmt.Sprintf("http://localhost%s", path)
	if len(query) > 0 {
		querystr := query.Encode()
		if querystr != "" {
			url = fmt.Sprintf("%s?%s", url, querystr)
		}
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		return
	}
	res, err = c.Client.Post(url, "application/json", bytes.NewReader(buf))
	if err != nil {
		return
	}
	body, err = ioutil.ReadAll(res.Body)
	return
}

type Response struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

type LsRes struct {
	Response
	List []core.Job `json:"list"`
}

func main() {
	sock := flag.String("sock", "/tmp/gocrond.sock", "Socket file")
	subCommand := flag.NewFlagSet("sub", flag.ExitOnError)
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}

	client := &HClient{
		Client: http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					// mo.Log(network, addr)
					return net.Dial("unix", *sock)
				},
			},
		},
	}

	switch args[0] {
	case "ping":
		_, body, err := client.Get("/ping", nil)
		if err != nil {
			mo.Panic(err)
		}
		mo.Info(string(body))
	case "ls":
		_, body, err := client.Get("/ls", nil)
		if err != nil {
			mo.Panic(err)
		}

		res := LsRes{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			mo.Panic(err)
		}
		if res.Ret != 0 {
			mo.Panicf("Error %s", res.Msg)
		}
		for i, v := range res.List {
			state := ""
			if v.Running {
				pid := color(fmt.Sprintf("%d", v.Pid), "34", "39")
				state = color(fmt.Sprintf("[Running, pid:%s]", pid), "32", "39")
			} else {
				state = color("[Waiting]", "2;37", "0;39")
			}
			index := color(fmt.Sprintf("%d", i+1), "32", "39")
			id := color(v.ID, fmt.Sprintf("38;5;%d", strHashCode(v.ID)), "39")
			runCnt := color(fmt.Sprintf("- Run %d times", v.RunCnt), "2", "22;0;39")
			fmt.Printf("%s %s %s %s %s %s %s\n", index, id, color(v.Spec, "2", "22;0;39"), color(v.Cmd, "33", "39"), state, color(v.Title, "2", "22;0;39"), runCnt)
		}
	case "exec":
		subCommand.Parse(args[1:])
		subArgs := subCommand.Args()
		_, body, err := client.Post("/exec", nil, subArgs[0])
		if err != nil {
			mo.Panic(err)
		}
		res := Response{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			mo.Panic(err)
		}
		if res.Ret != 0 {
			mo.Error(res.Msg)
		} else {
			mo.Successf("%s", res.Msg)
		}
	default:
		mo.Panicf("Unknown command: %s", args[0])
	}

	// quit := make(chan os.Signal, 1)
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// <-quit
	// mo.Info("Shutdown Server ...")
}
