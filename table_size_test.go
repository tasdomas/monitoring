// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring_test

import (
	"github.com/juju/postgrestest"
	jc "github.com/juju/testing/checkers"
	"github.com/prometheus/client_golang/prometheus"
	prometheusinternal "github.com/prometheus/client_model/go"
	gc "gopkg.in/check.v1"

	"github.com/cloud-green/monitoring"
)

var _ = gc.Suite(&tableSizeSuite{})

type tableSizeSuite struct {
	db *postgrestest.DB
}

func (s *tableSizeSuite) SetUpTest(c *gc.C) {
	db, err := postgrestest.New()
	c.Assert(err, jc.ErrorIsNil)
	s.db = db

	q := `CREATE TABLE tests (n int);`
	_, err = s.db.Exec(q)
	c.Assert(err, jc.ErrorIsNil)

	q = `INSERT INTO tests VALUES ($1)`
	for i := 1; i <= 20; i++ {
		_, err := s.db.Exec(q, i)
		c.Assert(err, jc.ErrorIsNil)
	}
}

func (s *tableSizeSuite) TearDownTest(c *gc.C) {
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
}

func (s *tableSizeSuite) TestCollector(c *gc.C) {
	m, err := monitoring.NewTableSizeCollector("test", s.db.DB, "tests")
	c.Assert(err, jc.ErrorIsNil)

	ch := make(chan prometheus.Metric, 10)
	m.Collect(ch)

	assertValue(c, ch, 20, "tests")

	// Add more rows to the table and check the monitored value again.
	q := `INSERT INTO tests VALUES ($1)`
	for i := 1; i <= 15; i++ {
		_, err := s.db.Exec(q, i)
		c.Assert(err, jc.ErrorIsNil)
	}

	m.Collect(ch)
	assertValue(c, ch, 35, "tests")

	// Remove rows from the table.
	q = `DELETE FROM tests WHERE n % 3 = 0`
	_, err = s.db.Exec(q)
	c.Assert(err, jc.ErrorIsNil)

	m.Collect(ch)
	assertValue(c, ch, 24, "tests")

}

func (s *tableSizeSuite) TestCollectorAllTables(c *gc.C) {
	m, err := monitoring.NewTableSizeCollector("test", s.db.DB)
	c.Assert(err, jc.ErrorIsNil)

	ch := make(chan prometheus.Metric, 10)
	m.Collect(ch)

	assertValue(c, ch, 20, "tests")
}

func assertValue(c *gc.C, ch chan prometheus.Metric, count float64, label string) {
	value := getValue(c, ch, label)
	c.Assert(value, gc.Equals, count)
}

func getValue(c *gc.C, ch chan prometheus.Metric, label string) float64 {
	var m prometheus.Metric
	var raw prometheusinternal.Metric
	select {
	case m = <-ch:
	default:
		c.Error("metric not provided by collector")
	}

	err := m.Write(&raw)
	c.Assert(err, jc.ErrorIsNil)

	labels := raw.GetLabel()
	// handle for different labeling of table/collection size monitors.
	// The table or collection name is the last label in any case.
	l := len(labels)
	c.Assert(labels[l-1].GetValue(), gc.Equals, label)

	cnt := raw.GetGauge()
	value := cnt.GetValue()
	return value
}
