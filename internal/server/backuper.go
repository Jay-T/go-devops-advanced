package server

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/Jay-T/go-devops.git/internal/utils/metric"
)

// StorageBackuper interfaces describes a storage for metrics.
type StorageBackuper interface {
	SaveMetric(ctx context.Context, mMap map[string]metric.Metric) error
	RestoreMetrics(ctx context.Context, mMap map[string]metric.Metric) error
	CheckStorageStatus(ctx context.Context) error
}

// DBStorageBackuper backs up metrics to DB.
type DBStorageBackuper struct {
	db *sql.DB
}

// SaveMetric saves metrics to storage (DB).
func (dbBackuper *DBStorageBackuper) SaveMetric(ctx context.Context, mMap map[string]metric.Metric) error {
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
		_, err = stmt.ExecContext(ctx, metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			log.Println(err)
			errRollback := tx.Rollback()
			if errRollback != nil {
				log.Println(errRollback)
				return errRollback
			}
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
func (dbBackuper *DBStorageBackuper) RestoreMetrics(ctx context.Context, mMap map[string]metric.Metric) error {
	recs := make([]metric.Metric, 0)
	query := `
		SELECT * FROM metrics
	`
	rows, err := dbBackuper.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	for rows.Next() {
		var rec metric.Metric
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
func (dbBackuper *DBStorageBackuper) CheckStorageStatus(ctx context.Context) error {
	if err := dbBackuper.db.PingContext(ctx); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// FileStorageBackuper backs up metrics to a file.
type FileStorageBackuper struct {
	filename string
}

// SaveMetric saves metrics to storage (file).
func (fileBackuper *FileStorageBackuper) SaveMetric(ctx context.Context, mMap map[string]metric.Metric) error {
	var MetricList []metric.Metric
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	producer, err := NewProducer(fileBackuper.filename, flags)
	if err != nil {
		log.Fatal(err)
	}

	for _, metric := range mMap {
		MetricList = append(MetricList, metric)
	}
	if err = producer.WriteMetric(&MetricList); err != nil {
		log.Fatal(err)
	}
	err = producer.Close()
	if err != nil {
		return err
	}
	return nil
}

// RestoreMetrics restores metrics from storage (file).
func (fileBackuper *FileStorageBackuper) RestoreMetrics(ctx context.Context, mMap map[string]metric.Metric) error {
	flags := os.O_RDONLY | os.O_CREATE
	consumer, err := NewConsumer(fileBackuper.filename, flags)
	if err != nil {
		return err
	}
	err = consumer.ReadEvents(mMap)
	if err != nil {
		return err
	}
	return nil
}

// CheckStorageStatus checks nothing here. Interface requirement.
func (fileBackuper *FileStorageBackuper) CheckStorageStatus(ctx context.Context) error {
	return nil
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
			log.Print(dbBackuper.db)
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
