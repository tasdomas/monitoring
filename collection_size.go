// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring

import (
	"fmt"
	"sync"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	log = loggo.GetLogger("monitoring.collectors")
)

type stats struct {
	Count int32 `bson:"count"`
	Size  int32 `bson:"size"`
}

type statsGetter struct {
	lock    sync.Mutex
	session *mgo.Session
	closed  bool
}

// NewStatsGetter returns a new new statsGetter used for
// getting Count and Size information about a collection
// and closing the session.
func NewStatsGetter(s *mgo.Session) *statsGetter {
	return &statsGetter{
		session: s,
	}
}

// Get returns the Count and Size information of the given collection.
// If the underlying session is already closed it returns an error.
func (s *statsGetter) Get(db, coll string) (st stats, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed {
		return st, errors.New("session is closed")
	}
	defer func() {
		if r := recover(); r != nil {
			s.closed = true
			log.Warningf("recovered from panic: %v", r)
			err = errors.New("session is closed")
		}
	}()
	session := s.session.Copy()
	defer session.Close()

	collection := session.DB(db).C(coll)
	err = collection.Database.Run(bson.M{"collStats": collection.Name}, &st)
	if err != nil {
		return st, errors.Trace(err)
	}
	return st, nil
}

// Close closes the underlying mgo session.
func (s *statsGetter) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.session.Close()
	s.closed = true
}

// CollectionSizeCollector implements the prometheus.Collector interface and
// reports the size of the specified mongo collection.
type CollectionSizeCollector struct {
	sizeDesc  *prometheus.Desc
	countDesc *prometheus.Desc

	statsGetter    *statsGetter
	database       string
	collectionName string
}

// NewCollectionSizeCollector returns a new collection size collector with specified properties.
func NewCollectionSizeCollector(namespace, subsystem, namePrefix, database, collectionName string, getter *statsGetter) *CollectionSizeCollector {
	return &CollectionSizeCollector{
		sizeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, fmt.Sprintf("%s_collection_size_bytes", namePrefix)),
			fmt.Sprintf("%v collection size in bytes", collectionName),
			nil,
			nil),
		countDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, fmt.Sprintf("%s_collection_count", namePrefix)),
			fmt.Sprintf("%v collection object count", collectionName),
			nil,
			nil),
		database:       database,
		collectionName: collectionName,
		statsGetter:    getter,
	}
}

// Describe implements the prometheus.Collector interface.
func (u *CollectionSizeCollector) Describe(c chan<- *prometheus.Desc) {
	c <- u.sizeDesc
	c <- u.countDesc
}

// Collect implements the prometheus.Collector interface.
func (u *CollectionSizeCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := u.statsGetter.Get(u.database, u.collectionName)
	if err != nil {
		log.Errorf("failed to report %v collection stats: %v", u.collectionName, err)
		return
	}

	mSize, err := prometheus.NewConstMetric(u.sizeDesc, prometheus.GaugeValue, float64(stats.Size))
	if err != nil {
		log.Errorf("failed to report %v collection stats: %v", u.collectionName, err)
		return
	}
	mCount, err := prometheus.NewConstMetric(u.countDesc, prometheus.GaugeValue, float64(stats.Count))
	if err != nil {
		log.Errorf("failed to report %v collection stats: %v", u.collectionName, err)
		return
	}
	ch <- mSize
	ch <- mCount
}
