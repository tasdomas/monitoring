// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package monitoring

import (
	"fmt"

	"github.com/juju/loggo"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	log = loggo.GetLogger("collectors")
)

// CollectionSizeCollector implements the prometheus.Collector interface and
// reports the size of the specified mongo collection.
type CollectionSizeCollector struct {
	sizeDesc  *prometheus.Desc
	countDesc *prometheus.Desc

	session        *mgo.Session
	database       string
	collectionName string
}

// NewCollectionSizeCollector returns a new collection size collector with specified properties.
func NewCollectionSizeCollector(namespace, subsystem, namePrefix, database, collectionName string, s *mgo.Session) *CollectionSizeCollector {
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
		session:        s,
		database:       database,
		collectionName: collectionName,
	}
}

// Describe implements the prometheus.Collector interface.
func (u *CollectionSizeCollector) Describe(c chan<- *prometheus.Desc) {
	c <- u.sizeDesc
	c <- u.countDesc
}

// Collect implements the prometheus.Collector interface.
func (u *CollectionSizeCollector) Collect(ch chan<- prometheus.Metric) {
	session := u.session.Copy()
	defer session.Close()

	collection := session.DB(u.database).C(u.collectionName)
	var stats struct {
		Count int32 `bson:"count"`
		Size  int32 `bson:"size"`
	}
	err := collection.Database.Run(bson.M{"collStats": collection.Name}, &stats)
	if err != nil {
		log.Errorf("failed to report %v collection stats: %v", collection.Name, err)
		return
	}

	mSize, err := prometheus.NewConstMetric(u.sizeDesc, prometheus.GaugeValue, float64(stats.Size))
	if err != nil {
		log.Errorf("failed to report %v collection stats: %v", collection.Name, err)
		return
	}
	mCount, err := prometheus.NewConstMetric(u.countDesc, prometheus.GaugeValue, float64(stats.Count))
	if err != nil {
		log.Errorf("failed to report %v collection stats: %v", collection.Name, err)
		return
	}
	ch <- mSize
	ch <- mCount
}
