// Application for receiving metrics over WEB and storing in DB.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Jay-T/go-devops.git/internal/server"
	_ "github.com/lib/pq"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := server.GetConfig()
	if err != nil {
		log.Fatal("Error while getting config.", err.Error())
	}

	backuper, err := server.NewBackuper(ctx, cfg)
	if err != nil {
		log.Print("Error during StorageBackuper initialization.")
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	if cfg.GRPC {
		fmt.Println("GRPC SERVICE")
	} else {
		HTTPServer, err := server.NewHTTPService(ctx, cfg, backuper)
		if err != nil {
			log.Fatalf("Could not run HTTP server. Error: %s", err)
		}
		go HTTPServer.StartServer(ctx, backuper)
		<-sigChan
		HTTPServer.StopServer(ctx, cancel, backuper)
	}
}
