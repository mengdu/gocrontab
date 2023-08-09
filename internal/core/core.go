package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
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

// func parseLine(line string) []string {
// 	args := strings.Fields(strings.TrimSpace(line))
// 	for i := 0; i < len(args); i++ {
// 		args[i] = strings.Trim(args[i], "\"")
// 	}
// 	return args
// }

type Job struct {
	ID      string        `json:"id"`
	Title   string        `json:"title"`
	Spec    string        `json:"spec"`
	Cmd     string        `json:"cmd"`
	Running bool          `json:"running"`
	PrevUse time.Duration `json:"prev_use"`
	cronId  cron.EntryID
}

func (j *Job) Exec(isManual bool) {
	j.Running = true
	start := time.Now()
	defer func() {
		j.Running = false
		j.PrevUse = time.Since(start)
	}()
	log := mo.WithTag(j.ID)
	log.Debugf("Exec: %s, manual: %t", j.Cmd, isManual)
	cmd := exec.Command("bash", "-c", j.Cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		log.Errorf("Error: %s", err)
	}
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
	// mo.Infof("Jobs: %v", jobs)
	for i := 0; i < len(jobs); i++ {
		job := jobs[i]
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
	m.jobs = jobs
	return nil
}

func (m *Manager) GetJobs() []Job {
	return m.jobs
}

func (m *Manager) Exec(id string) {
	for i := 0; i < len(m.jobs); i++ {
		if m.jobs[i].ID == id {
			m.jobs[i].Exec(true)
			return
		}
	}
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
			arr = append(arr, Job{
				ID:    hex.EncodeToString(hash[:]),
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
