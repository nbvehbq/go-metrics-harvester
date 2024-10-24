package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
	"github.com/nbvehbq/go-metrics-harvester/internal/metric"
	"github.com/nbvehbq/go-metrics-harvester/internal/storage"
	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func TestPostgres_Set(t *testing.T) {
	tests := []struct {
		name    string
		value   metric.Metric
		wantErr bool
	}{
		{
			name:    "set gauge",
			value:   metric.Metric{ID: "one", MType: metric.Gauge, Value: ptr(54.0)},
			wantErr: false,
		},
		{
			name:    "set counter",
			value:   metric.Metric{ID: "two", MType: metric.Counter, Delta: ptr[int64](42)},
			wantErr: false,
		},
		{
			name:    "set invalid type",
			value:   metric.Metric{ID: "three", MType: metric.Counter, Value: ptr(54.0)},
			wantErr: true,
		},
	}

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	st := &Storage{
		sqlx.NewDb(db, "sqlmock"),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				mock.ExpectExec(`INSERT INTO metric`).
					WithArgs(tt.value.ID, tt.value.MType, tt.value.Delta, tt.value.Value).
					WillReturnError(storage.ErrMetricMalformed)
			} else {
				mock.ExpectExec(`INSERT INTO metric`).
					WithArgs(tt.value.ID, tt.value.MType, tt.value.Delta, tt.value.Value).
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			err := st.Set(context.Background(), tt.value)
			assert.Equal(t, tt.wantErr, err != nil)

			if tt.wantErr {
				assert.Equal(t, storage.ErrMetricMalformed, err)
			} else {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestPostgres_Get(t *testing.T) {
	tests := []struct {
		name string
		want metric.Metric
		res  bool
	}{
		{
			name: "get gauge",
			want: metric.Metric{ID: "one", MType: metric.Gauge, Value: ptr(54.0)},
			res:  true,
		},
	}

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	st := &Storage{
		sqlx.NewDb(db, "sqlmock"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"}).
				AddRow(tt.want.ID, tt.want.MType, tt.want.Delta, tt.want.Value)
			mock.ExpectQuery(`SELECT id, mtype, delta, value FROM metric`).
				WithArgs(tt.want.ID).
				WillReturnRows(rows)

			res, ok := st.Get(context.Background(), tt.want.ID)
			assert.Equal(t, ok, tt.res)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestPostgres_List(t *testing.T) {
	tests := []struct {
		name string
		want []metric.Metric
		res  bool
	}{
		{
			name: "get gauge",
			want: []metric.Metric{
				{ID: "one", MType: metric.Gauge, Value: ptr(54.0)},
			},
			res: true,
		},
	}

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	st := &Storage{
		sqlx.NewDb(db, "sqlmock"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"})
			for _, v := range tt.want {
				rows.AddRow(v.ID, v.MType, v.Delta, v.Value)
			}
			mock.ExpectQuery(`SELECT id, mtype, delta, value FROM metric`).
				WillReturnRows(rows)

			res, err := st.List(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}
