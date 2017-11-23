// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring

import (
	"database/sql"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// NewTableSizeCollector returns a new collector that monitors table sizes.
func NewTableSizeCollector(namespace string, db *sql.DB) (*dbTableSizeCollector, error) {
	var dbName string
	q := `SELECT current_database();`
	err := db.QueryRow(q).Scan(&dbName)
	if err != nil {
		return nil, err
	}
	return &dbTableSizeCollector{
		countDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "database", fmt.Sprintf("%s_table_row_count", dbName)),
			"table row count",
			[]string{"table"},
			nil),
		db:     db,
		dbName: dbName,
	}, nil
}

type dbTableSizeCollector struct {
	countDesc *prometheus.Desc

	db     *sql.DB
	dbName string
}

var _ prometheus.Collector = (*dbTableSizeCollector)(nil)

// Describe implements the prometheus.Collector interface.
func (u *dbTableSizeCollector) Describe(c chan<- *prometheus.Desc) {
	c <- u.countDesc
}

// Collect implements the prometheus.Collector interface.
func (u *dbTableSizeCollector) Collect(ch chan<- prometheus.Metric) {
	var tables []string
	var tableName string
	tableQuery := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';`
	rows, err := u.db.Query(tableQuery)
	if err != nil {
		log.Errorf("failed to query existing tables: %v", err)
	}
	for rows.Next() {
		err = rows.Scan(&tableName)
		if err != nil {
			rows.Close()
			log.Errorf("failed to scan defined table names: %v", err)
		}
		tables = append(tables, tableName)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		log.Errorf("failed to scan defined table names: %v", err)
	}
	if len(tables) == 0 {
		log.Warningf("no tables found on DB %q", u.dbName)
		return
	}
	for _, tableName := range tables {
		var rows int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)

		if err := u.db.QueryRow(query).Scan(&rows); err != nil {
			log.Errorf("failed to query table size for %q: %v", tableName, err)
			return
		}

		mCount, err := prometheus.NewConstMetric(u.countDesc, prometheus.GaugeValue, float64(rows), tableName)
		if err != nil {
			log.Errorf("failed to report table size for %q: %v", tableName, err)
			return
		}
		ch <- mCount
	}
}
