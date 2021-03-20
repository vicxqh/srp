package main

import (
	"fmt"
	"os"

	"github.com/vicxqh/srp/agent/internal"

	flag "github.com/spf13/pflag"
	"github.com/vicxqh/srp/log"
)

var (
	logPath     string
	logLevel    string
	name        string
	description string
	server      string
)

func init() {
	flag.StringVar(&logPath, "log", "", "log file path.")
	flag.StringVar(&logLevel, "log-level", "info", "log level.[info|debug|warning|error]")
	hostname, _ := os.Hostname()
	flag.StringVar(&name, "name", hostname, "agent name(id)")
	flag.StringVar(&description, "description", "", "more detailed description about this agent")
	flag.StringVar(&server, "server", "", "srp server address")
}

func main() {
	flag.Parse()
	err := log.Init(logPath)
	if err != nil {
		fmt.Println("failed to init log file,", err)
		os.Exit(1)
	}
	log.SetLevelString(logLevel)
	if len(server) == 0 {
		fmt.Println("server address is required")
		os.Exit(1)
	}

	internal.ConnectToServer(server, name, description)
}
