//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2023 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

package schema

/// TODO-RAfT START
// Currently, our schema definition relies on auto-generated Swagger definitions for “entities/models/*”.
// While Swagger definitions work ok for transport, they introduce inefficiencies when used in core logic for several reasons.
// Types are not explicitly defined, leading to unnecessary parsing, even for primitive types.
// This places an undue burden on garbage collection, which has to manage numerous unnecessary pointers.

// In this task, our goal is to develop an efficient data structure, a native core data structure that addresses these deficiencies.
// To achieve this, we need to identify the various access patterns for entities/models to optimize the data structure for these scenarios.

/// TODO-RAFT END

import (
	"github.com/weaviate/weaviate/entities/models"
	"github.com/weaviate/weaviate/entities/vectorindex/hnsw"
	sharding "github.com/weaviate/weaviate/usecases/sharding/config"
)

type Collection struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`

	// inverted index config
	InvertedIndexConfig InvertedIndexConfig `json:"invertedIndex,omitempty"`

	// TODO-RAFT START
	// Can we also get rid of the interface{} in the value side ?
	// Configuration specific to modules this Weaviate instance has installed
	ModuleConfig map[string]interface{} `json:"moduleConfig,omitempty"`
	// TODO-RAFT END

	// multi tenancy config
	MultiTenancyConfig MultiTenancyConfig `json:"multiTenancyConfig,omitempty"`

	// The properties of the class.
	Properties []Property `json:"properties"`

	// replication config
	ReplicationConfig ReplicationConfig `json:"replicationConfig,omitempty"`

	// Manage how the index should be sharded and distributed in the cluster
	ShardingConfig ShardingConfig `json:"shardingConfig,omitempty"`

	// VectorIndexType which vector index to use
	VectorIndexType VectorIndexType `json:"vectorIndexType,omitempty"`
	// TODO-RAFT START
	// TODO-RAFT: can we do better?
	// We can unmarshal vector index config into a concrete struct
	// depending on VectorIndexType and store into VectorIndexConfig

	// Vector-index config, that is specific to the type of index selected in vectorIndexType
	VectorIndexConfig VectorIndexConfig `json:"vectorIndexConfig,omitempty"`
	// TODO-RAFT END

	// Specify how the vectors for this class should be determined. The options are either 'none' - this means you have to import a vector with each object yourself - or the name of a module that provides vectorization capabilities, such as 'text2vec-contextionary'. If left empty, it will use the globally configured default which can itself either be 'none' or a specific module.
	Vectorizer string `json:"vectorizer,omitempty"`
}

type VectorIndexType int

const (
	VectorIndexTypeHNSW VectorIndexType = iota
	// VectorIndexTypeOther is mostly used in test where we have fake index type
	VectorIndexTypeOther
	// VectorIndexTypeUnknown is used when we parse an unexpected index type
	VectorIndexTypeUnknown
)

var (
	vectorIndexTypeToString = map[VectorIndexType]string{
		VectorIndexTypeHNSW:    "hnsw",
		VectorIndexTypeOther:   "other",
		VectorIndexTypeUnknown: "unknown",
	}
	stringToVectorIndexType = map[string]VectorIndexType{
		"hnsw":    VectorIndexTypeHNSW,
		"other":   VectorIndexTypeOther,
		"unknown": VectorIndexTypeUnknown,
	}
)

type MultiTenancyConfig struct {
	Enabled bool `json:"enabled"`
}

type ReplicationConfig struct {
	// Factor represent replication factor
	Factor int64 `json:"factor,omitempty"`
}

type ShardingConfig struct {
	VirtualPerPhysical  int    `json:"virtualPerPhysical"`
	DesiredCount        int    `json:"desiredCount"`
	ActualCount         int    `json:"actualCount"`
	DesiredVirtualCount int    `json:"desiredVirtualCount"`
	ActualVirtualCount  int    `json:"actualVirtualCount"`
	Key                 string `json:"key"`
	Strategy            string `json:"strategy"`
	Function            string `json:"function"`
}

type Property struct {
	// Name of the property as URI relative to the schema URL.
	Name string `json:"name,omitempty"`

	// Description of the property.
	Description string `json:"description,omitempty"`

	// Can be a reference to another type when it starts with a capital (for example Person), otherwise "string" or "int".
	// TODO-RAFT: Can we make DataType a slice of interface where other type and native type implements it ?
	DataType []string `json:"data_type"`

	// Optional. Should this property be indexed in the inverted index. Defaults to true. If you choose false, you will not be able to use this property in where filters. This property has no affect on vectorization decisions done by modules
	IndexFilterable bool `json:"indexFilterable,omitempty"`

	// Optional. Should this property be indexed in the inverted index. Defaults to true. If you choose false, you will not be able to use this property in where filters, bm25 or hybrid search. This property has no affect on vectorization decisions done by modules (deprecated as of v1.19; use indexFilterable or/and indexSearchable instead)
	IndexInverted bool `json:"indexInverted,omitempty"`

	// Optional. Should this property be indexed in the inverted index. Defaults to true. Applicable only to properties of data type text and text[]. If you choose false, you will not be able to use this property in bm25 or hybrid search. This property has no affect on vectorization decisions done by modules
	IndexSearchable bool `json:"indexSearchable,omitempty"`

	// Configuration specific to modules this Weaviate instance has installed
	ModuleConfig map[string]interface{} `json:"moduleConfig,omitempty"`

	// The properties of the nested object(s). Applies to object and object[] data types.
	NestedProperties []NestedProperty `json:"nestedProperties,omitempty"`

	// Determines tokenization of the property as separate words or whole field. Optional. Applies to text and text[] data types. Allowed values are `word` (default; splits on any non-alphanumerical, lowercases), `lowercase` (splits on white spaces, lowercases), `whitespace` (splits on white spaces), `field` (trims). Not supported for remaining data types
	// Enum: [word lowercase whitespace field]
	Tokenization string `json:"tokenization,omitempty"`
}

type NestedProperty struct {
	// name
	Name string `json:"name,omitempty"`
	// description
	Description string `json:"description,omitempty"`
	// data type
	DataType []string `json:"data_type"`

	// index filterable
	IndexFilterable bool `json:"index_filterable,omitempty"`

	// index searchable
	IndexSearchable bool `json:"index_searchable,omitempty"`

	// nested properties
	NestedProperties []NestedProperty `json:"nested_properties,omitempty"`

	// tokenization
	// Enum: [word lowercase whitespace field]
	Tokenization string `json:"tokenization,omitempty"`
}

func (n *NestedProperty) FromModel(m *models.NestedProperty) {
	n.DataType = m.DataType
	n.Description = m.Description
	if m.IndexFilterable != nil {
		n.IndexFilterable = *m.IndexFilterable
	}
	if m.IndexSearchable != nil {
		n.IndexSearchable = *m.IndexSearchable
	}
	n.Name = m.Name
	n.Tokenization = m.Tokenization
	for _, npm := range m.NestedProperties {
		np := NestedProperty{}
		(&np).FromModel(npm)

		n.NestedProperties = append(n.NestedProperties, np)
	}
}

func (n *NestedProperty) ToModel() *models.NestedProperty {
	return nil
}

func (p *Property) FromModel(m *models.Property) {
	p.DataType = m.DataType
	p.Description = m.Description
	if m.IndexFilterable != nil {
		p.IndexFilterable = *m.IndexFilterable
	}
	if m.IndexInverted != nil {
		p.IndexInverted = *m.IndexInverted
	}
	if m.IndexSearchable != nil {
		p.IndexSearchable = *m.IndexSearchable
	}
	if v, ok := m.ModuleConfig.(map[string]interface{}); ok {
		p.ModuleConfig = v
	}
	p.Name = m.Name
	p.Tokenization = m.Tokenization
	for _, npm := range m.NestedProperties {
		np := NestedProperty{}
		(&np).FromModel(npm)

		p.NestedProperties = append(p.NestedProperties, np)
	}
}

func (p *Property) ToModel() *models.Property {
	return nil
}

func (s *ShardingConfig) FromModel(m interface{}) {
	v, ok := m.(sharding.Config)
	if !ok {
		return
	}
	s.ActualCount = v.ActualCount
	s.ActualVirtualCount = v.ActualVirtualCount
	s.DesiredCount = v.DesiredCount
	s.DesiredVirtualCount = v.DesiredVirtualCount
	s.Function = v.Function
	s.Key = v.Key
	s.Strategy = v.Strategy
	s.VirtualPerPhysical = v.VirtualPerPhysical
}

func (s *ShardingConfig) ToModel() interface{} {
	var m sharding.Config

	m.ActualCount = s.ActualCount
	m.ActualVirtualCount = s.ActualVirtualCount
	m.DesiredCount = s.DesiredCount
	m.DesiredVirtualCount = s.DesiredVirtualCount
	m.Function = s.Function
	m.Key = s.Key
	m.Strategy = s.Strategy
	m.VirtualPerPhysical = s.VirtualPerPhysical

	return m
}

func (c *Collection) FromClass(m models.Class) {
	c.Name = m.Class
	c.Description = m.Description
	c.InvertedIndexConfig.FromModel(m.InvertedIndexConfig)
	if v, ok := m.ModuleConfig.(map[string]interface{}); ok {
		c.ModuleConfig = v
	}
	if m.MultiTenancyConfig != nil {
		c.MultiTenancyConfig.Enabled = m.MultiTenancyConfig.Enabled
	}
	c.Properties = make([]Property, len(m.Properties))
	for i, mp := range m.Properties {
		p := Property{}
		(&p).FromModel(mp)

		c.Properties[i] = p
	}
	if m.ReplicationConfig != nil {
		c.ReplicationConfig.Factor = m.ReplicationConfig.Factor
	}
	if m.ShardingConfig != nil {
		c.ShardingConfig.FromModel(m.ShardingConfig)
	}

	c.VectorIndexType = stringToVectorIndexType[m.VectorIndexType]
	switch c.VectorIndexType {
	case VectorIndexTypeHNSW:
		c.VectorIndexConfig = m.VectorIndexConfig.(hnsw.UserConfig)
	case VectorIndexTypeOther:
		// Unmarshal into default struct
	case VectorIndexTypeUnknown:
		// Do not unmarshal
	}
	// Vectorizer
}

func (c *Collection) ToClass() models.Class {
	var m models.Class

	return m
}
