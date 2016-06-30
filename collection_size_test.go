// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring_test

import (
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	"github.com/prometheus/client_golang/prometheus"
	prometheusinternal "github.com/prometheus/client_model/go"
	gc "gopkg.in/check.v1"
	"gopkg.in/mgo.v2/bson"

	"github.com/cloud-green/monitoring"
)

var _ = gc.Suite(&collectionSizeSuite{})

var _ prometheus.Collector = (*monitoring.CollectionSizeCollector)(nil)

type collectionSizeSuite struct {
	testing.MgoSuite
}

func (s *collectionSizeSuite) TestCollectionSizeReporting(c *gc.C) {
	u := monitoring.NewCollectionSizeCollector("test", "test", "test", "test", "test_collection", s.Session)

	collection := s.Session.DB("test").C("test_collection")
	err := collection.Insert(bson.M{"test": true})
	c.Assert(err, jc.ErrorIsNil)

	ch := make(chan prometheus.Metric, 2)

	u.Collect(ch)
	var m prometheus.Metric
	// read the size
	select {
	case m = <-ch:
	default:
		c.Error("metric not provided by collector")
	}

	var raw prometheusinternal.Metric
	err = m.Write(&raw)
	c.Assert(err, jc.ErrorIsNil)

	cnt := raw.GetGauge()
	valueOne := cnt.GetValue()

	// read the count
	select {
	case m = <-ch:
	default:
		c.Error("metric not provided by collector")
	}

	err = m.Write(&raw)
	c.Assert(err, jc.ErrorIsNil)

	cnt = raw.GetGauge()
	val := cnt.GetValue()
	c.Assert(val, gc.Equals, float64(1.0))

	err = collection.Insert(bson.M{"test": true})
	c.Assert(err, jc.ErrorIsNil)

	u.Collect(ch)
	// read the size
	select {
	case m = <-ch:
	default:
		c.Error("metric not provided by collector")
	}

	err = m.Write(&raw)
	c.Assert(err, jc.ErrorIsNil)

	cnt = raw.GetGauge()
	valueTwo := cnt.GetValue()
	c.Assert(2*valueOne, gc.Equals, valueTwo)

	// read the count
	select {
	case m = <-ch:
	default:
		c.Error("metric not provided by collector")
	}

	err = m.Write(&raw)
	c.Assert(err, jc.ErrorIsNil)

	cnt = raw.GetGauge()
	val = cnt.GetValue()
	c.Assert(val, gc.Equals, float64(2.0))
}
