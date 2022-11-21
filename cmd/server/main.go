// Application for receiving metrics over WEB and storing in DB.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jay-T/go-devops.git/internal/server"
	_ "github.com/lib/pq"
)

var (
	address       *string
	restore       *bool
	storeInterval *time.Duration
	storeFile     *string
	s             server.Service
	key           *string
	db            *string
)

func main() {
	s, err := server.NewService()
	if err != nil {
		log.Fatalf("Could not load agent config. Error: %s", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	if s.Cfg.DB != "" {
		s.DB, err = sql.Open("postgres", s.Cfg.DB)
		if err != nil {
			log.Println(err)
		}
		defer s.DB.Close()
		s.DBInit(ctx)
	}

	go s.StartServer(ctx)

	if s.Cfg.Restore {
		if s.DB != nil {
			log.Printf("Restoring metrics from DB")
			if err := s.RestoreMetricFromDB(ctx); err != nil {
				log.Println(err)
			}
		} else if s.Cfg.StoreFile != "" {
			log.Printf("Restoring metrics from file '%s'", s.Cfg.StoreFile)
			if err := s.RestoreMetricFromFile(); err != nil {
				log.Println(err)
			}
		}
	}

	if s.Cfg.StoreFile != "" && s.Cfg.StoreInterval > time.Duration(0) {
		log.Printf("Saving results to file with interval %s", s.Cfg.StoreInterval)
		go s.StartRecordInterval(ctx)
	}

	<-sigChan
	if s.DB != nil {
		err := s.SaveMetricToDB(ctx)
		if err != nil {
			log.Print(err.Error())
		}
	} else {
		s.SaveMetricToFile()
	}
	cancel()
	server.CloseApp()
}
