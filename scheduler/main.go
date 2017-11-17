package main

import (
	"flag"
	"log"
	"os"

	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/app"
)

func main() {
	cfg := app.NewConfig()
	fs := flag.NewFlagSet("scheduler", flag.ExitOnError)
	cfg.AddFlags(fs)
	fs.Parse(os.Args[1:])

	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
