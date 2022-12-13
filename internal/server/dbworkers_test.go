package server

import (
	"context"
	"log"
	"testing"

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

func TestNewServiceDB(t *testing.T) {
	ctx := context.TODO()
	_, err := NewServiceDB(ctx, "teststring")
	assert.NoError(t, err)
}
