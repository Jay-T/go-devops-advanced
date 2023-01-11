// Application collects system metrics and sends to server.
package main

import (
	"fmt"
	"log"
	_ "net/http/pprof"

	"github.com/Jay-T/go-devops.git/internal/agent"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)

	cfg, err := agent.GetConfig()
	if err != nil {
		log.Fatal("Error while getting config.", err.Error())
	}
	if cfg.GRPC {
		a, err := agent.NewGRPCAgent(cfg)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to create gRPC agent: %w", err))
		}
		a.Run()
	} else {
		a, err := agent.NewHTTPAgent(cfg)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to create gRPC agent: %w", err))
		}
		a.Run()
	}
}
