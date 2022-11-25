package main

import (
	"context"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jay-T/go-devops.git/internal/agent"
)

func main() {
	a, err := agent.NewAgent()
	if err != nil {
		log.Fatalf("Could not load agent config. Error: %s", err)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	dataChan := make(chan agent.Data, 50)
	syncChan := make(chan time.Time)

	ctx, cancel := context.WithCancel(context.Background())
	go a.RunTicker(ctx, syncChan)
	go a.NewMetric(ctx, dataChan)
	go a.GetDataByInterval(ctx, dataChan, syncChan)
	go a.GetMemDataByInterval(ctx, dataChan, syncChan)
	go a.GetCPUDataByInterval(ctx, dataChan)
	go a.SendDataByInterval(ctx, dataChan)
	a.StopAgent(sigChan, cancel)
}
