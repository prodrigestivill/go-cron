package gocron

import (
	"github.com/robfig/cron/v3"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type LastRun struct {
	Exit_status  int
	Stdout       string
	Stderr       string
	ExitTime     string
	Pid          int
	StartingTime string
}

type CurrentState struct {
	Running  map[string]*LastRun
	Last     *LastRun
	Schedule string
}

var Current_state CurrentState

func copyOutput(out *string, src io.ReadCloser, pid int) {
	buf := make([]byte, 1024)
	for {
		n, err := src.Read(buf)
		if n != 0 {
			s := string(buf[:n])
			*out = *out + s
			log.Printf("%d: %v", pid, s)
		}
		if err != nil {
			break
		}
	}
}

func execute(command string, args []string) {

	cmd := exec.Command(command, args...)

	run := new(LastRun)
	run.StartingTime = time.Now().Format(time.RFC3339)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start: %v", err)
	}

	run.Pid = cmd.Process.Pid
	Current_state.Running[strconv.Itoa(run.Pid)] = run

	go copyOutput(&run.Stdout, stdout, run.Pid)
	go copyOutput(&run.Stderr, stderr, run.Pid)

	log.Println(run.Pid, "cmd:", command, strings.Join(args, " "))

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			// so set the error code to tremporary value
			run.Exit_status = 127
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				run.Exit_status = status.ExitStatus()
				log.Printf("%d Exit Status: %d", run.Pid, run.Exit_status)
			}
		} else {
			log.Fatalf("cmd.Wait: %v", err)
		}
	}

	run.ExitTime = time.Now().Format(time.RFC3339)

	delete(Current_state.Running, strconv.Itoa(run.Pid))
	//run.Pid = 0
	Current_state.Last = run
}

func Create(schedule string, command string, args []string) (cr *cron.Cron, wgr *sync.WaitGroup, err error) {
	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	
	// Validate schedule
	if _, err := parser.Parse(schedule); err != nil {
		return nil, nil, err
	}

	wg := &sync.WaitGroup{}

	c := cron.New(cron.WithParser(parser))
	Current_state = CurrentState{map[string]*LastRun{}, &LastRun{}, schedule}
	log.Println("new cron:", schedule)

	c.AddFunc(schedule, func() {
		wg.Add(1)
		execute(command, args)
		wg.Done()
	})

	return c, wg, nil
}

func Start(c *cron.Cron) {
	c.Start()
}

func Stop(c *cron.Cron, wg *sync.WaitGroup) {
	log.Println("Stopping")
	c.Stop()
	log.Println("Waiting")
	wg.Wait()
	log.Println("Exiting")
	os.Exit(0)
}

func RunJobs(c *cron.Cron) {
	for _, e := range c.Entries() {
		e.Job.Run()
	}
}
