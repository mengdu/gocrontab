package core

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/mengdu/mo"
	"github.com/robfig/cron/v3"
)

func New() *Manager {
	return &Manager{
		cron: cron.New(cron.WithLocation(time.Local)),
		// cron: cron.New(cron.WithSeconds()),
	}
}

var Set = wire.NewSet(New)

type Job struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Spec    string        `json:"spec"`
	Cmd     string        `json:"cmd"`
	Running bool          `json:"running"`
	Pid     int           `json:"pid"`
	RunCnt  int           `json:"run_cnt"`
	PrevUse time.Duration `json:"prev_use"`
	cronId  cron.EntryID
}

type LWrite struct {
	Log func(log []byte)
}

func (w *LWrite) Write(c []byte) (n int, err error) {
	w.Log(bytes.TrimRight(c, "\n"))
	return len(c), nil
}

func (j *Job) Exec(isManual bool) {
	if j.Running {
		mo.Errorf("Job %s is already running", j.ID)
		return
	}
	j.Running = true
	j.RunCnt++
	start := time.Now()
	log := mo.WithTag(j.ID)
	log.Logf("Start exec %s, manual: %t, times: %d", j.Cmd, isManual, j.RunCnt)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", j.Cmd)
	} else {
		cmd = exec.Command("bash", "-c", j.Cmd)
	}
	cmd.Stderr = &LWrite{Log: func(msg []byte) {
		log.Error(string(msg))
	}}
	cmd.Stdout = &LWrite{Log: func(msg []byte) {
		log.Log(string(msg))
	}}
	if err := cmd.Start(); err != nil {
		log.Errorf("Error: %s", err)
	}
	go func() {
		j.Pid = cmd.Process.Pid
		if err := cmd.Wait(); err != nil {
			log.Error(err)
		}
		j.Running = false
		j.Pid = 0
		j.PrevUse = time.Since(start)
		log.Logf("Exec finish, use %s", j.PrevUse)
	}()
}

type Manager struct {
	cron *cron.Cron
	jobs []Job
}

func (m *Manager) Start(file string) error {
	jobs, err := parseCrontab(file)
	if err != nil {
		return err
	}
	m.jobs = jobs
	// mo.Infof("Jobs: %v", jobs)
	for i := 0; i < len(jobs); i++ {
		job := &jobs[i]
		id, err := m.cron.AddFunc(job.Spec, func() {
			job.Exec(false)
		})
		if err != nil {
			return err
		}
		job.cronId = id
		mo.Debugf("Added job: %#v", job)
	}
	m.cron.Start()
	mo.Infof("Initiated %d jobs", len(jobs))
	return nil
}

func (m *Manager) GetJobs() []Job {
	return m.jobs
}

func (m *Manager) Exec(id string) error {
	for i := 0; i < len(m.jobs); i++ {
		if m.jobs[i].ID == id {
			if m.jobs[i].Running {
				return fmt.Errorf("`%s` job is already running", id)
			}
			m.jobs[i].Exec(true)
			return nil
		}
	}
	return fmt.Errorf("Job not found: %s", id)
}

func parseCrontab(file string) ([]Job, error) {
	buf, err := ioutil.ReadFile(file)
	arr := []Job{}
	if err != nil {
		return arr, err
	}
	lines := strings.Split(string(buf), "\n")
	title := ""
	index := uint(0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		re := regexp.MustCompile(`^(([0-9*\/\-,]+ +?){5})(.*)$`)
		if len(line) > 0 && line[0] == '#' {
			line = strings.TrimSpace(line[1:])
			if !re.Match([]byte(line)) {
				title = line
			}
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			spec := strings.TrimSpace(matches[1])
			script := strings.TrimSpace(matches[3])
			index++
			hash := md5.Sum([]byte(fmt.Sprintf("%d:%s", index, line)))
			id := hex.EncodeToString(hash[:])
			arr = append(arr, Job{
				ID:    id[len(id)-6:],
				Title: title,
				Spec:  spec,
				Cmd:   script,
			})
			if title != "" {
				title = ""
			}
		}
	}
	return arr, nil
}
