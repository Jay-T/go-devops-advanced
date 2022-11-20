package main

import (
	"context"
	"log"
)

func (s Service) DBInit(ctx context.Context) error {
	const qry = `
		CREATE TABLE IF NOT EXISTS metrics (
			id text PRIMARY KEY,
			mtype text NOT NULL,
			delta bigint,
			value double precision
		)`
	if _, err := s.db.ExecContext(ctx, qry); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (s Service) RestoreMetricFromDB(ctx context.Context) error {
	recs := make([]Metric, 0)
	qry := `
		SELECT * FROM metrics
	`
	rows, err := s.db.QueryContext(ctx, qry)
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
		metrics[i.ID] = i
	}
	return nil
}

func (s Service) SaveMetricToDB(ctx context.Context) error {
	addRecord := `
		INSERT INTO metrics (id, mtype, delta, value) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET delta = $3,
			value = $4
	`
	for _, metric := range metrics {
		_, err := s.db.ExecContext(ctx, addRecord, metric.ID, metric.MType, metric.Delta, metric.Value)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func (s Service) saveListToDB(ctx context.Context, mList *[]Metric) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	addRecord := `
		INSERT INTO metrics (id, mtype, delta, value) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE 
		SET delta = $3,
			value = $4
	`
	stmt, err := tx.PrepareContext(ctx, addRecord)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, m := range *mList {
		switch m.MType {
		case counter:
			if metrics[m.ID].Delta == nil {
				metrics[m.ID] = m
			} else {
				*metrics[m.ID].Delta += *m.Delta
			}
		case gauge:
			metrics[m.ID] = m
		}
		if _, err = stmt.ExecContext(ctx, m.ID, m.MType, metrics[m.ID].Delta, metrics[m.ID].Value); err != nil {
			return err
		}
	}
	return tx.Commit()
}
