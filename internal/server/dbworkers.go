package server

import (
	"context"
	"database/sql"

	"github.com/Jay-T/go-devops.git/internal/utils/metric"
)

func (s *GenericService) saveListToDB(ctx context.Context, mList *[]metric.Metric) error {
	for _, m := range *mList {
		switch m.MType {
		case counter:
			if s.Metrics[m.ID].Delta == nil {
				s.Metrics[m.ID] = m
			} else {
				*s.Metrics[m.ID].Delta += *m.Delta
			}
		case gauge:
			s.Metrics[m.ID] = m
		}
	}
	if err := s.backuper.SaveMetric(ctx, s.Metrics); err != nil {
		return err
	}
	return nil
}

// NewServiceDB returns a DB connection for service.
func NewServiceDB(ctx context.Context, dbAddress string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbAddress)
	if err != nil {
		return nil, err
	}

	return db, nil
}
