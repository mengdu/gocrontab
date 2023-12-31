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
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/mengdu/mo"
	"github.com/olekukonko/tablewriter"
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

type Job struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Spec    string        `json:"spec"`
	Cmd     string        `json:"cmd"`
	Running bool          `json:"running"`
	Pid     int           `json:"pid"`
	RunCnt  int           `json:"run_cnt"`
	PrevUse time.Duration `json:"prev_use"`
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
	List []Job `json:"list"`
	Info struct {
		File    string    `json:"file"`
		StartAt time.Time `json:"start_at"`
	} `json:"info"`
}

func main() {
	sockFile := flag.String("sock", "", "Socket file")
	subCommand := flag.NewFlagSet("sub", flag.ExitOnError)
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
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

	client := &HClient{
		Client: http.Client{
			Transport: &http.Transport{
				Dial: func(network, addr string) (net.Conn, error) {
					// mo.Log(network, addr)
					return net.Dial("unix", sock)
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

		fmt.Printf("Total %s jobs\n", color(fmt.Sprintf("%d", len(res.List)), "34", "39"))
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"no", "id", "cron", "status", "times", "title", "cmd"})
		table.SetCenterSeparator("+")
		table.SetColumnSeparator("│")
		table.SetRowSeparator("─")
		table.SetRowLine(true)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		for i, v := range res.List {
			state := ""
			if v.Running {
				state = color(fmt.Sprintf("%d", v.Pid), "32", "39")
			} else {
				state = color("-", "2", "22;0;39")
			}
			id := color(v.ID, fmt.Sprintf("38;5;%d", strHashCode(v.ID)), "39")
			runCnt := color(fmt.Sprintf("%d", v.RunCnt), "34", "39")
			table.Append([]string{fmt.Sprintf("%d", i+1), id, v.Spec, state, runCnt, v.Title, v.Cmd})
		}
		table.Render()
		fmt.Print(color(fmt.Sprintf("\nCron file: %s\nStart at: %s\n", res.Info.File, res.Info.StartAt.Format(time.RFC3339)), "2", "22;0;39"))
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
