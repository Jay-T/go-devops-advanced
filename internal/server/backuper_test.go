package server

import (
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Jay-T/go-devops.git/internal/utils/metric"
	"github.com/stretchr/testify/assert"
)

func TestDBInit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		err = db.Close()
		if err != nil {
			log.Print(err)
		}
	}()
	ctx := context.TODO()

	dbs := &DBStorageBackuper{
		db: db,
	}

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(1, 1))
	err = dbs.DBInit(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(".*").WillReturnError(New("TestError"))
	err = dbs.DBInit(ctx)
	assert.Error(t, err)

}

func TestRestoreMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		err = db.Close()
		if err != nil {
			log.Print(err)
		}
	}()
	ctx := context.TODO()

	dbs := &DBStorageBackuper{
		db: db,
	}

	s := GenericService{
		Metrics: map[string]metric.Metric{},
	}

	rs := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"}).AddRow("Alloc", "gauge", "0", "23456")

	mock.ExpectQuery(`SELECT \* FROM metrics`).
		WillDelayFor(1 * time.Second).
		WillReturnRows(rs)

	err = dbs.RestoreMetrics(ctx, s.Metrics)
	if err != nil {
		log.Print(err)
	}
	assert.Equal(t, s.Metrics["Alloc"], metric.Metric{
		ID:    "Alloc",
		MType: gauge,
		Delta: getIntPointer(0),
		Value: getFloatPointer(23456),
	})
}

func TestSaveMetricToDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer func() {
		err = db.Close()
		if err != nil {
			log.Print(err)
		}
	}()
	ctx := context.TODO()

	dbs := &DBStorageBackuper{
		db: db,
	}

	s := GenericService{
		Metrics: map[string]metric.Metric{},
	}

	s.Metrics = map[string]metric.Metric{
		"Alloc": {
			ID:    "Alloc",
			MType: gauge,
			Delta: getIntPointer(0),
			Value: getFloatPointer(23456),
		},
	}

	metric := s.Metrics["Alloc"]
	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().
		WithArgs(metric.ID, metric.MType, metric.Delta, metric.Value).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	err = dbs.SaveMetric(ctx, s.Metrics)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().
		WithArgs(metric.ID, metric.MType, metric.Delta, metric.Value).
		WillReturnError(New("TestError"))
	mock.ExpectRollback()
	err = dbs.SaveMetric(ctx, s.Metrics)
	assert.Error(t, err)
}

func TestNewBackuper(t *testing.T) {
	cfg := &Config{
		DBAddress: "postgress://address.test/?sslmode=disable",
	}
	ctx := context.TODO()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	fakeOpen := func(str1 string, str2 string) (*sql.DB, error) {
		return db, nil
	}
	patch := monkey.Patch(sql.Open, fakeOpen)
	defer patch.Unpatch()

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS metrics .*`).WillReturnResult(sqlmock.NewResult(1, 1))

	_, err = NewBackuper(ctx, cfg)
	assert.NoError(t, err)

	cfg = &Config{
		DBAddress: "",
		StoreFile: "/tmp/test",
	}

	_, err = NewBackuper(ctx, cfg)
	assert.NoError(t, err)
}
