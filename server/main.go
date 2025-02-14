package main

import (
	"fmt"
	"os"

	"github.com/vicxqh/srp/server/internal"

	flag "github.com/spf13/pflag"
	"github.com/vicxqh/srp/log"
)

var (
	logPath  string
	logLevel string
	httpPort int
	dataPort int
)

func init() {
	flag.StringVar(&logPath, "log", "", "log file path.")
	flag.StringVar(&logLevel, "log-level", "info", "log level.[info|debug|warning|error]")
	flag.IntVar(&httpPort, "http", 8010, "http service port")
	flag.IntVar(&dataPort, "data", 8011, "data forwarding port")
}

func main() {
	flag.Parse()
	err := log.Init(logPath)
	if err != nil {
		fmt.Println("failed to init log file,", err)
		os.Exit(1)
	}
	log.SetLevelString(logLevel)

	s := internal.NewServer(httpPort, dataPort)
	s.Run()
}
