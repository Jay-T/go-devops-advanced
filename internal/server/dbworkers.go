package server

import (
	"context"
	"database/sql"
)

func (s *Service) saveListToDB(ctx context.Context, mList *[]Metric, backuper StorageBackuper) error {
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
		if err := backuper.SaveMetric(ctx, s.Metrics); err != nil {
			return err
		}
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
