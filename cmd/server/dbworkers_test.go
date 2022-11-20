package main

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// type DBMock struct {
// 	throwError bool
// }

// func (db DBMock) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
// 	if db.throwError {
// 		return nil, New("TestError")
// 	}
// 	return nil, nil
// }

// func (db DBMock) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
// 	rows := &sql.Rows{}
// 	return rows, nil
// }

// func (db DBMock) Begin() (*sql.Tx, error) {
// 	return nil, nil
// }

// func (db DBMock) Close() error {
// 	return nil
// }
// func (db DBMock) PingContext(ctx context.Context) error {
// 	return nil
// }

// type DBRow struct{}

// func (r DBRow) Close() error {
// 	return nil
// }

// func (r DBRow) Columns() ([]string, error) {
// 	return nil, nil
// }

// func (r DBRow) Err() error {
// 	return nil
// }

// func (r DBRow) Next() bool {
// 	return true
// }

// func (r DBRow) Scan(dest ...interface{}) error {
// 	return nil
// }

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

func (mm MetricMatcher) Match(v driver.Value) bool {
	_, ok := v.(string)
	_, ok = v.(*int64)
	_, ok = v.(*float64)
	return ok
}

func TestDBInit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx := context.TODO()

	s := Service{
		db: db,
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
		db: db,
	}

	rs := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"}).AddRow("Alloc", "gauge", "0", "23456")

	mock.ExpectQuery(`SELECT \* FROM metrics`).
		WillDelayFor(1 * time.Second).
		WillReturnRows(rs)

	err = s.RestoreMetricFromDB(ctx)
	assert.Equal(t, metrics["Alloc"], Metric{
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
		db: db,
	}

	metrics = map[string]Metric{
		"Alloc": {
			ID:    "Alloc",
			MType: gauge,
			Delta: getIntPointer(0),
			Value: getFloatPointer(23456),
		},
	}

	metric := metrics["Alloc"]
	mock.ExpectExec(`INSERT INTO metrics`).
		WithArgs(metric.ID, metric.MType, metric.Delta, metric.Value).
		WillReturnResult(sqlmock.NewResult(1, 1))
	err = s.SaveMetricToDB(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(`INSERT INTO metrics`).
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
		db: db,
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
