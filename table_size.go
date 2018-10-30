// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring

import (
	"context"

	"database/sql"
	"fmt"

	"github.com/juju/zaputil"
	"github.com/juju/zaputil/zapctx"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const rowCountCutoff = 10000.0

// NewTableSizeCollector returns a new collector that monitors table sizes.
// If the list of tables to monitor is not specified, all tables in the schema will be monitored.
func NewTableSizeCollector(namespace string, db *sql.DB, tables ...string) (*dbTableSizeCollector, error) {
	var dbName string
	err := db.QueryRow(`SELECT current_database()`).Scan(&dbName)
	if err != nil {
		return nil, err
	}
	var schemaName = "public" // use 'public' schema by default
	var schemaValue sql.NullString
	err = db.QueryRow(`SELECT current_schema()`).Scan(&schemaValue)
	if err != nil {
		return nil, err
	}
	if schemaValue.Valid {
		schemaName = schemaValue.String
	}
	return &dbTableSizeCollector{
		countDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "database", "table_row_count"),
			"table row count",
			[]string{"database", "table"},
			nil),
		db:         db,
		dbName:     dbName,
		schemaName: schemaName,
		tables:     tables,
	}, nil
}

type dbTableSizeCollector struct {
	countDesc *prometheus.Desc

	db         *sql.DB
	dbName     string
	schemaName string

	tables []string
}

var _ prometheus.Collector = (*dbTableSizeCollector)(nil)

// Describe implements the prometheus.Collector interface.
func (u *dbTableSizeCollector) Describe(c chan<- *prometheus.Desc) {
	c <- u.countDesc
}

// Collect implements the prometheus.Collector interface.
func (u *dbTableSizeCollector) Collect(ch chan<- prometheus.Metric) {
	logger := zapctx.Logger(context.Background()).With(zap.String("db", u.dbName))
	// Collecting table sizes is done in two steps. First table row count
	// estimates are queried, because this is fast.
	// Then for tables whose row count estimate is below the threshold,
	// an exact query is issued.

	tables := map[string]float64{}
	var tableName string
	var rowEstimate float64

	var rows *sql.Rows
	var err error

	if len(u.tables) > 0 {
		tableEstimateQuery := `SELECT t.table_name, c.reltuples 
        FROM information_schema.tables t INNER JOIN pg_class c
            ON c.relname = t.table_name 
            WHERE t.table_schema=$1 
             AND t.table_type='BASE TABLE'
             AND t.table_name = ANY($2)`

		rows, err = u.db.Query(tableEstimateQuery, u.schemaName, pq.Array(u.tables))
		if err != nil {
			logger.Error("failed to query existing tables", zaputil.Error(err))
			return
		}
	} else {
		tableEstimateQuery := `SELECT t.table_name, c.reltuples 
        FROM information_schema.tables t INNER JOIN pg_class c
            ON c.relname = t.table_name 
            WHERE t.table_schema=$1 
             AND t.table_type='BASE TABLE'`

		rows, err = u.db.Query(tableEstimateQuery, u.schemaName)
		if err != nil {
			logger.Error("failed to query existing tables", zaputil.Error(err))
			return
		}
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&tableName, &rowEstimate)
		if err != nil {
			logger.Error("failed to scan defined table names", zaputil.Error(err))
			return
		}
		tables[tableName] = rowEstimate
	}
	if err = rows.Err(); err != nil {
		logger.Error("failed to scan defined table names", zaputil.Error(err))
		return
	}
	if len(tables) == 0 {
		logger.Warn("no tables found on DB")
		return
	}
	for tableName, rowEstimate := range tables {
		// If the table's row count estimate is more than the cutoff value,
		// report the estimate.
		if rowEstimate > rowCountCutoff {
			mCount, err := prometheus.NewConstMetric(u.countDesc, prometheus.GaugeValue, rowEstimate,
				u.dbName, tableName)
			if err != nil {
				logger.Error("failed to report table size", zap.String("table", tableName), zaputil.Error(err))
				return
			}
			ch <- mCount
			continue
		}
		var rows int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)

		if err := u.db.QueryRow(query).Scan(&rows); err != nil {
			logger.Error("failed to query table size", zap.String("table", tableName), zaputil.Error(err))
			return
		}

		mCount, err := prometheus.NewConstMetric(u.countDesc, prometheus.GaugeValue, float64(rows),
			u.dbName, tableName)
		if err != nil {
			logger.Error("failed to report table size", zap.String("table", tableName), zaputil.Error(err))
			return
		}
		ch <- mCount
	}
}
