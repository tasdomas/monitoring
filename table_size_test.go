// Copyright 2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring_test

import (
	"github.com/juju/postgrestest"
	jc "github.com/juju/testing/checkers"
	"github.com/prometheus/client_golang/prometheus"
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
	m, err := monitoring.NewTableSizeCollector("test", s.db.DB)
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
