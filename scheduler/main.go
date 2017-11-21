package main

import (
	"flag"
	"log"
	"os"

	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/core"
)

func main() {
	cfg := core.NewConfig()
	fs := flag.NewFlagSet("scheduler", flag.ExitOnError)
	cfg.AddFlags(fs)
	fs.Parse(os.Args[1:])

	if err := core.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
