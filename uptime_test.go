// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring_test

import (
	"time"

	jc "github.com/juju/testing/checkers"
	"github.com/prometheus/client_golang/prometheus"
	prometheusinternal "github.com/prometheus/client_model/go"
	gc "gopkg.in/check.v1"

	"github.com/cloud-green/monitoring"
)

type UptimeSuite struct{}

var _ = gc.Suite(&UptimeSuite{})

var _ prometheus.Collector = (*monitoring.UptimeCollector)(nil)

func (t *UptimeSuite) TestUptimeReporting(c *gc.C) {
	now := time.Now()
	u, err := monitoring.NewUptimeCollector("test", "test", "test", time.Now)
	c.Assert(err, jc.ErrorIsNil)
	ch := make(chan prometheus.Metric, 1000)
	u.Collect(ch)
	var m prometheus.Metric
	select {
	case m = <-ch:
	default:
		c.Error("metric not provided by collector")
	}

	var raw prometheusinternal.Metric
	err = m.Write(&raw)
	c.Assert(err, jc.ErrorIsNil)

	cnt := raw.GetCounter()
	val := cnt.GetValue()
	c.Assert(val, gc.Equals, float64(now.Unix()))
}
