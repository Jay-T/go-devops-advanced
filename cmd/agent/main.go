// Application collects system metrics and sends to server.
package main

import (
	"context"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/Jay-T/go-devops.git/internal/agent"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := agent.GetConfig()
	if err != nil {
		log.Fatal("Error while getting config.", err.Error())
	}

	agent, err := agent.NewAgent(cfg)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create gRPC agent: %w", err))
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	doneChan := make(chan struct{})

	go agent.Run(ctx, doneChan)

	<-sigChan
	agent.StopAgent(sigChan, doneChan, cancel)
}
