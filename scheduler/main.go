package main

import (
	"flag"
	"os"

	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/core"
	log "github.com/sirupsen/logrus"
	"github.com/teo/logrus-prefixed-formatter"
)

func init() {
	log.SetFormatter(&prefixed.TextFormatter{
		FullTimestamp:	true,
		SpacePadding:	20,
		PrefixPadding:	12,
	})
}

func main() {
	cfg := core.NewConfig()
	fs := flag.NewFlagSet("scheduler", flag.ExitOnError)
	cfg.AddFlags(fs)
	fs.Parse(os.Args[1:])

	if err := core.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
