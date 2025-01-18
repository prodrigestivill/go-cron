package main

import (
	"flag"
	"github.com/prodrigestivill/go-cron/gocron"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var build string
var version string

func main() {
	flagArgs, execArgs := splitArgs()
	os.Args = flagArgs

	var (
		help     = flag.Bool("h", false, "display usage")
		port     = flag.String("p", "18080", "bind healthcheck to a specific port, set to 0 to not open HTTP port at all")
		schedule = flag.String("s", "* * * * *", "schedule the task the cron style")
		initrun  = flag.Bool("i", false, "run one first time on start after initialization")
	)

	flag.Parse()

	if *help {
		println("Usage of", os.Args[0], "(build", build, ")")
		println(os.Args[0], " [ OPTIONS ] -- [ COMMAND ]")
		flag.PrintDefaults()
		os.Exit(1)
	}
	log.Println("Running version:", version)

	c, wg, err := gocron.Create(*schedule, execArgs[0], execArgs[1:len(execArgs)])
	if err != nil {
		log.Fatalf("Invalid schedule: %v", err)
	}

	go gocron.Start(c)
	if *port != "0" {
		go gocron.Http_server(*port)
	}
	if *initrun {
		go gocron.RunJobs(c)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	println(<-ch)
	gocron.Stop(c, wg)
}

func splitArgs() (flagArgs []string, execArgs []string) {

	split := len(os.Args)

	for idx, e := range os.Args {

		if e == "--" {
			split = idx
			break
		}

	}

	flagArgs = os.Args[0:split]

	if split < len(os.Args) {
		execArgs = os.Args[split+1 : len(os.Args)]
	} else {
		execArgs = []string{}
	}

	return flagArgs, execArgs

}
