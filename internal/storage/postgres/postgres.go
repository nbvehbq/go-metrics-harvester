package postgres

import (
	"context"
	"encoding/json"
	"io"

	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
	"github.com/nbvehbq/go-metrics-harvester/internal/logger"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Storage struct {
	db *sqlx.DB
}

func NewStorage(ctx context.Context, DSN string) (*Storage, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", DSN)
	if err != nil {
		return nil, errors.Wrap(err, "connect to db")
	}

	if err := initDatabaseStructure(ctx, db); err != nil {
		return nil, errors.Wrap(err, "init db")
	}

	return &Storage{db: db}, nil
}

func NewFrom(ctx context.Context, src io.Reader, DSN string) (*Storage, error) {
	db, err := sqlx.ConnectContext(ctx, "postgres", DSN)
	if err != nil {
		return nil, errors.Wrap(err, "connect to db")
	}

	if err := initDatabaseStructure(ctx, db); err != nil {
		return nil, errors.Wrap(err, "init db")
	}

	if err := clearDatabase(ctx, db); err != nil {
		return nil, errors.Wrap(err, "clear db")
	}

	var list []metric.Metric
	if err := json.NewDecoder(src).Decode(&list); err != nil {
		return nil, err
	}

	// values := make([]string, 0, len(list))
	// valuesArgs := make([]interface{}, 0, len(list)*4)
	// for _, v := range list {
	// 	values = append(values, "($1, $2, $3, $4)")
	// 	valuesArgs = append(valuesArgs, v.ID)
	// 	valuesArgs = append(valuesArgs, v.MType)
	// 	valuesArgs = append(valuesArgs, v.Delta)
	// 	valuesArgs = append(valuesArgs, v.Value)
	// }
	// fmt.Sprintf("INSERT INTO metric (id, mtype, delta, value) VALUES %s", fmt.Sprintf())

	tx, err := db.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "begin transaction")
	}

	stmt, err := tx.Prepare(pq.CopyIn("metric", "id", "mtype", "delta", "value"))
	if err != nil {
		return nil, errors.Wrap(err, "prepare")
	}

	for _, m := range list {
		_, err = stmt.Exec(m.ID, m.MType, m.Delta, m.Value)
		if err != nil {
			return nil, errors.Wrap(err, "exec item")
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, errors.Wrap(err, "exec all")
	}

	err = stmt.Close()
	if err != nil {
		return nil, errors.Wrap(err, "close")
	}

	err = tx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "commit")
	}

	return &Storage{db: db}, nil
}

func clearDatabase(ctx context.Context, db *sqlx.DB) error {
	_, err := db.ExecContext(ctx, `truncate table public."metric";`)
	if err != nil {
		return err
	}

	return nil
}

func initDatabaseStructure(ctx context.Context, db *sqlx.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS "metric" (
	  id TEXT NOT NULL,
	  mtype TEXT NOT NULL,
	  delta INT,
	  value DOUBLE PRECISION,

		CONSTRAINT "id_pkey" PRIMARY KEY ("id")
	);`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) Set(value metric.Metric) error {
	if value.MType == metric.Counter && value.Delta == nil {
		return storage.ErrMetricMalformed
	}

	if value.MType == metric.Gauge && value.Value == nil {
		return storage.ErrMetricMalformed
	}

	query := `
	INSERT INTO metric (id, mtype, delta, value)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT(id)
	DO UPDATE SET 
		delta = EXCLUDED.delta + metric.delta, 
		value = EXCLUDED.value;`
	if _, err := s.db.Exec(query, value.ID, value.MType, value.Delta, value.Value); err != nil {
		return errors.Wrap(err, "insert metric")
	}

	return nil
}

func (s *Storage) Get(key string) (metric.Metric, bool) {
	var res metric.Metric
	err := s.db.Get(&res, `SELECT id, mtype, delta, value FROM metric WHERE id = $1;`, key)
	if err != nil {
		logger.Log.Error("get metric", zap.Error(err))
	}

	return res, err == nil
}

func (s *Storage) List() ([]metric.Metric, error) {
	var res []metric.Metric
	err := s.db.Select(&res, `SELECT id, mtype, delta, value FROM metric;`)
	if err != nil {
		logger.Log.Error("select metric", zap.Error(err))
		return res, errors.Wrap(err, "select metric")
	}

	return res, nil
}

func (s *Storage) Persist(dest io.Writer) error {
	list, err := s.List()
	if err != nil {
		return errors.Wrap(err, "persist")
	}

	if err := json.NewEncoder(dest).Encode(&list); err != nil {
		return errors.Wrap(err, "encode list")
	}

	return nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
