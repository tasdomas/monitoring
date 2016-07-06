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
	getter := monitoring.NewStatsGetter(s.Session)
	u := monitoring.NewCollectionSizeCollector("test", "test", "test", "test", "test_collection", getter)

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

func (s *collectionSizeSuite) TestCollectionOnClosedSession(c *gc.C) {
	session := s.Session.Copy()
	getter := monitoring.NewStatsGetter(session)
	getter.Close()
	u := monitoring.NewCollectionSizeCollector("test", "test", "test", "test", "test_collection", getter)

	collection := s.Session.DB("test").C("test_collection")
	err := collection.Insert(bson.M{"test": true})
	c.Assert(err, jc.ErrorIsNil)

	ch := make(chan prometheus.Metric, 2)

	u.Collect(ch)
	// read the size
	select {
	case <-ch:
		c.Fatalf("we expected no metric")
	default:
	}

	// read the count
	select {
	case <-ch:
		c.Fatalf("we expected no metric")
	default:
	}
}

// TestCollectionOnClosedSessionGraceful closes the session directly and checks
// we handle this cleanly without panicing.
func (s *collectionSizeSuite) TestCollectionOnClosedSessionGraceful(c *gc.C) {
	session := s.Session.Copy()
	getter := monitoring.NewStatsGetter(session)
	session.Close() // We close the session directly.
	u := monitoring.NewCollectionSizeCollector("test", "test", "test", "test", "test_collection", getter)

	collection := s.Session.DB("test").C("test_collection")
	err := collection.Insert(bson.M{"test": true})
	c.Assert(err, jc.ErrorIsNil)

	ch := make(chan prometheus.Metric, 2)

	u.Collect(ch)
	// read the size
	select {
	case <-ch:
		c.Fatalf("we expected no metric")
	default:
	}

	// read the count
	select {
	case <-ch:
		c.Fatalf("we expected no metric")
	default:
	}
}
