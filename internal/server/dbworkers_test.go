package server

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type errorTest struct {
	s string
}

func (e *errorTest) Error() string {
	return e.s
}

func New(text string) error {
	return &errorTest{text}
}

type MetricMatcher struct{}

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

	s := Service{
		Metrics: map[string]Metric{},
	}

	rs := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"}).AddRow("Alloc", "gauge", "0", "23456")

	mock.ExpectQuery(`SELECT \* FROM metrics`).
		WillDelayFor(1 * time.Second).
		WillReturnRows(rs)

	err = dbs.RestoreMetrics(ctx, s.Metrics)
	if err != nil {
		log.Print(err)
	}
	assert.Equal(t, s.Metrics["Alloc"], Metric{
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

	s := Service{
		Metrics: map[string]Metric{},
	}

	s.Metrics = map[string]Metric{
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

func TestSaveListToDB(t *testing.T) {
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

	mList := make([]Metric, 0, 43)
	mList = []Metric{
		{
			ID:    "Alloc",
			MType: gauge,
			Delta: getIntPointer(0),
			Value: getFloatPointer(23456),
		},
		{
			ID:    "PollCount",
			MType: counter,
			Delta: getIntPointer(5),
			Value: getFloatPointer(0),
		},
	}

	s := Service{
		Metrics: map[string]Metric{},
	}

	mock.ExpectBegin()
	stmt := mock.ExpectPrepare(`INSERT INTO metrics`)
	for i := 0; i < len(mList); i++ {
		stmt.ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()
	err = s.saveListToDB(ctx, &mList, dbs)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().WillReturnError(New("TestError"))
	mock.ExpectRollback()

	err = s.saveListToDB(ctx, &mList, dbs)
	assert.Error(t, err)
}
