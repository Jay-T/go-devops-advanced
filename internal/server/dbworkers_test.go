package server

import (
	"context"
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
	defer db.Close()
	ctx := context.TODO()

	s := Service{
		DB: db,
	}

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(1, 1))
	err = s.DBInit(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(".*").WillReturnError(New("TestError"))
	err = s.DBInit(ctx)
	assert.Error(t, err)

}

func TestRestoreMetricFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx := context.TODO()

	s := Service{
		DB:      db,
		Metrics: map[string]Metric{},
	}

	rs := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"}).AddRow("Alloc", "gauge", "0", "23456")

	mock.ExpectQuery(`SELECT \* FROM metrics`).
		WillDelayFor(1 * time.Second).
		WillReturnRows(rs)

	s.RestoreMetricFromDB(ctx)
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
	defer db.Close()
	ctx := context.TODO()

	s := Service{
		DB:      db,
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
	err = s.SaveMetricToDB(ctx)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().
		WithArgs(metric.ID, metric.MType, metric.Delta, metric.Value).
		WillReturnError(New("TestError"))
	err = s.SaveMetricToDB(ctx)
	assert.Error(t, err)
}

func TestSaveListToDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx := context.TODO()

	s := Service{
		DB:      db,
		Metrics: map[string]Metric{},
	}
	mList := []Metric{
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

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO metrics`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = s.saveListToDB(ctx, &mList)
	assert.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO metrics`).ExpectExec().WillReturnError(New("TestError"))
	mock.ExpectRollback()

	err = s.saveListToDB(ctx, &mList)
	assert.Error(t, err)
}
