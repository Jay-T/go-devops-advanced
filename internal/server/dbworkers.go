package server

import (
	"context"
	"database/sql"
	"log"
)

// DBInit creates table with specific structure if it is not created yet.
func (s Service) DBInit(ctx context.Context) error {
	const queryInitialTable = `
		CREATE TABLE IF NOT EXISTS metrics (
			id text PRIMARY KEY,
			mtype text NOT NULL,
			delta bigint,
			value double precision
		)`
	if _, err := s.DB.ExecContext(ctx, queryInitialTable); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// RestoreMetricFromDB loads metrics values from DB.
func (s Service) RestoreMetricFromDB(ctx context.Context) error {
	recs := make([]Metric, 0)
	query := `
		SELECT * FROM metrics
	`
	rows, err := s.DB.QueryContext(ctx, query)
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
		s.Metrics[i.ID] = i
	}
	return nil
}

// SaveMetricToDB saves metrics to DB.
func (s Service) SaveMetricToDB(ctx context.Context) error {
	addRecordQuery := `
		INSERT INTO metrics (id, mtype, delta, value) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET delta = $3,
			value = $4
	`
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, addRecordQuery)
	if err != nil {
		return err
	}
	for _, metric := range s.Metrics {
		_, err := stmt.ExecContext(ctx, metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

func (s Service) saveListToDB(ctx context.Context, mList *[]Metric) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	addRecordQuery := `
		INSERT INTO metrics (id, mtype, delta, value) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET delta = $3,
			value = $4
	`
	stmt, err := tx.PrepareContext(ctx, addRecordQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()
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
		if _, err = stmt.ExecContext(ctx, m.ID, m.MType, s.Metrics[m.ID].Delta, s.Metrics[m.ID].Value); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// NewServiceDB returns a DB connection for service.
func (s Service) NewServiceDB(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("postgres", s.Cfg.DBAddress)
	if err != nil {
		return nil, err
	}

	s.DB = db
	err = s.DBInit(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
