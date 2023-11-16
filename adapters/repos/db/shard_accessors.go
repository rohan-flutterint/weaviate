package db


import (
	"github.com/weaviate/weaviate/adapters/repos/db/indexcounter"
	"github.com/weaviate/weaviate/adapters/repos/db/inverted"
	"github.com/weaviate/weaviate/adapters/repos/db/lsmkv"
	"github.com/weaviate/weaviate/entities/schema"

	"github.com/weaviate/weaviate/adapters/repos/db/propertyspecific"
)

func (s *Shard) Queue() *IndexQueue {
	return s.queue
}

func (s *Shard) VectorIndex() VectorIndex {
	return s.vectorIndex
}

func (s *Shard) Versioner() *shardVersioner {
	return s.versioner
}


func (s *Shard) Index() *Index {
	return s.index
}

func (s *Shard) Name() string {
	return s.name
}

func (s *Shard) Store() *lsmkv.Store {
	return s.store
}

func (s *Shard) Counter() *indexcounter.Counter {
	return s.counter
}

 func (s *Shard) GetVectorIndex() VectorIndex {
 	return s.VectorIndex()
 }

 func (s *Shard) GetPropertyIndices() propertyspecific.Indices {
 	return s.propertyIndices
 }

func (s *Shard) GetPropertyLengthTracker() *inverted.JsonPropertyLengthTracker {
	return s.propLenTracker
}

 func (s *Shard) SetPropertyLengthTracker(tracker *inverted.JsonPropertyLengthTracker) {
 	s.propLenTracker = tracker
}

func (s *Shard) Metrics() *Metrics {
	return s.metrics
}

func (s *Shard) setFallbackToSearchable(fallback bool) {
	s.fallbackToSearchable = fallback
}

func (s *Shard) addJobToQueue(job job) {
	s.centralJobQueue <- job
}

func (s *Shard) hasGeoIndex() bool {
	s.propertyIndicesLock.RLock()
	defer s.propertyIndicesLock.RUnlock()

	for _, idx := range s.propertyIndices {
		if idx.Type == schema.DataTypeGeoCoordinates {
			return true
		}
	}
	return false
}