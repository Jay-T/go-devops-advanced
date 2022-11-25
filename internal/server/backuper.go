package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"
)

// StorageBackuper interfaces describes a storage for metrics.
type StorageBackuper interface {
	SaveMetric(ctx context.Context, mMap map[string]Metric) error
	RestoreMetrics(ctx context.Context, mMap map[string]Metric) error
	CheckStorageStatus(w http.ResponseWriter, r *http.Request)
}

// DBStorageBackuper backs up metrics to DB.
type DBStorageBackuper struct {
	db *sql.DB
}

// SaveMetric saves metrics to storage (DB).
func (dbBackuper *DBStorageBackuper) SaveMetric(ctx context.Context, mMap map[string]Metric) error {
	addRecordQuery := `
		INSERT INTO metrics (id, mtype, delta, value) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET delta = $3,
			value = $4
	`
	tx, err := dbBackuper.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, addRecordQuery)
	if err != nil {
		return err
	}
	for _, metric := range mMap {
		_, err := stmt.ExecContext(ctx, metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

// RestoreMetrics restores metrics from storage (DB).
func (dbBackuper *DBStorageBackuper) RestoreMetrics(ctx context.Context, mMap map[string]Metric) error {
	recs := make([]Metric, 0)
	query := `
		SELECT * FROM metrics
	`
	rows, err := dbBackuper.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rec Metric
		err = rows.Scan(&rec.ID, &rec.MType, &rec.Delta, &rec.Value)
		if err != nil {
			return err
		}

		recs = append(recs, rec)

		err = rows.Err()
		if err != nil {
			return err
		}
	}
	for _, i := range recs {
		mMap[i.ID] = i
	}
	return nil
}

// DBInit creates table with specific structure if it is not created yet.
func (dbBackuper *DBStorageBackuper) DBInit(ctx context.Context) error {
	const queryInitialTable = `
		CREATE TABLE IF NOT EXISTS metrics (
			id text PRIMARY KEY,
			mtype text NOT NULL,
			delta bigint,
			value double precision
		)`
	if _, err := dbBackuper.db.ExecContext(ctx, queryInitialTable); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// CheckStorageStatus checks DB connection.
// URI: "/ping".
func (dbBackuper *DBStorageBackuper) CheckStorageStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := dbBackuper.db.PingContext(ctx); err != nil {
		log.Println(err)
		http.Error(w, "Error in DB connection.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// FileStorageBackuper backs up metrics to a file.
type FileStorageBackuper struct {
	filename string
}

// SaveMetric saves metrics to storage (file).
func (fileBackuper *FileStorageBackuper) SaveMetric(ctx context.Context, mMap map[string]Metric) error {
	var MetricList []Metric
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	producer, err := NewProducer(fileBackuper.filename, flags)
	if err != nil {
		log.Fatal(err)
	}

	for _, metric := range mMap {
		MetricList = append(MetricList, metric)
	}
	if err := producer.WriteMetric(&MetricList); err != nil {
		log.Fatal(err)
	}
	producer.Close()
	return nil
}

// RestoreMetrics restores metrics from storage (file).
func (fileBackuper *FileStorageBackuper) RestoreMetrics(ctx context.Context, mMap map[string]Metric) error {
	flags := os.O_RDONLY | os.O_CREATE
	consumer, err := NewConsumer(fileBackuper.filename, flags)
	if err != nil {
		return err
	}
	consumer.ReadEvents(mMap)
	return nil
}

// CheckStorageStatus checks nothing here. Interface requirement.
// URI: "/ping".
func (fileBackuper *FileStorageBackuper) CheckStorageStatus(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("The storage file is pretty fine."))
	w.WriteHeader(http.StatusOK)
}

// NewBackuper returns a new backuper instance.
func NewBackuper(ctx context.Context, cfg *Config) (StorageBackuper, error) {
	var backuper StorageBackuper

	if cfg.DBAddress != "" {
		dbBackuper := &DBStorageBackuper{}
		db, err := NewServiceDB(ctx, cfg.DBAddress)
		if err != nil {
			log.Print("Error during DB connection.")
			log.Fatal(err)
		}

		dbBackuper.db = db
		err = dbBackuper.DBInit(ctx)
		if err != nil {
			return nil, err
		}
		backuper = dbBackuper
	} else {
		fileBackuper := &FileStorageBackuper{
			filename: cfg.StoreFile,
		}
		backuper = fileBackuper
	}
	return backuper, nil
}
