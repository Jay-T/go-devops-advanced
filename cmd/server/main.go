// Application for receiving metrics over WEB and storing in DB.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Jay-T/go-devops.git/internal/server"
	_ "github.com/lib/pq"
)

func main() {
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

	s, err := server.NewService(ctx, cfg, backuper)
	if err != nil {
		log.Fatalf("Could not load agent config. Error: %s", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go s.StartServer(ctx, backuper)

	<-sigChan
	s.StopServer(ctx, cancel, backuper)
}
