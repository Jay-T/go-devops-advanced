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
	"time"

	"github.com/Jay-T/go-devops.git/internal/agent"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
	a, err := agent.NewAgent()
	if err != nil {
		log.Fatalf("Could not load agent config. Error: %s", err)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	dataChan := make(chan agent.Data)
	syncChan := make(chan time.Time)
	doneChan := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	go a.RunTicker(ctx, syncChan)
	go a.NewMetric(ctx, dataChan)
	go a.GetDataByInterval(ctx, dataChan, syncChan)
	go a.GetMemDataByInterval(ctx, dataChan, syncChan)
	go a.GetCPUDataByInterval(ctx, dataChan)
	go a.SendDataByInterval(ctx, dataChan, doneChan)
	a.StopAgent(sigChan, doneChan, cancel)
	// go a.SendDataByInterval(ctx, dataChan)
	// a.StopAgent(sigChan, cancel)
}
