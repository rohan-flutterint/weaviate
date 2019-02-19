/*                          _       _
 *__      _____  __ ___   ___  __ _| |_ ___
 *\ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
 * \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
 *  \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
 *
 * Copyright © 2016 - 2019 Weaviate. All rights reserved.
 * LICENSE: https://github.com/creativesoftwarefdn/weaviate/blob/develop/LICENSE.md
 * DESIGN & CONCEPT: Bob van Luijt (@bobvanluijt)
 * CONTACT: hello@creativesoftwarefdn.org
 */
package janusgraph

import (
	"github.com/creativesoftwarefdn/weaviate/database/schema"
	"github.com/creativesoftwarefdn/weaviate/database/schema/kind"
	"github.com/creativesoftwarefdn/weaviate/gremlin"
	"github.com/creativesoftwarefdn/weaviate/models"
	"github.com/go-openapi/strfmt"
)

func (j *Janusgraph) addClass(k kind.Kind, className schema.ClassName, UUID strfmt.UUID, atContext string, creationTimeUnix int64, lastUpdateTimeUnix int64, rawProperties interface{}) error {
	vertexLabel := j.state.GetMappedClassName(className)
	sourceClassAlias := "classToBeAdded"

	q := gremlin.G.AddV(string(vertexLabel)).
		As(sourceClassAlias).
		StringProperty(PROP_KIND, k.Name()).
		StringProperty(PROP_UUID, UUID.String()).
		StringProperty(PROP_CLASS_ID, string(vertexLabel)).
		StringProperty(PROP_AT_CONTEXT, atContext).
		Int64Property(PROP_CREATION_TIME_UNIX, creationTimeUnix).
		Int64Property(PROP_LAST_UPDATE_TIME_UNIX, lastUpdateTimeUnix)

	q, err := j.addEdgesToQuery(q, k, className, rawProperties, sourceClassAlias)
	if err != nil {
		return err
	}

	_, err = j.client.Execute(q)

	return err
}

const MaximumBatchItemsPerQuery = 50

type batchChunk struct {
	thing *models.Thing
	uuid  strfmt.UUID
}

func (j *Janusgraph) addThingsBatch(things []*models.Thing, uuids []strfmt.UUID) error {
	chunkSize := MaximumBatchItemsPerQuery
	chunks := len(things) / chunkSize
	if len(things) < chunkSize {
		chunks = 1
	}
	chunked := make([][]batchChunk, chunks)
	chunk := 0

	for i := 0; i < len(things); i++ {
		if i%chunkSize == 0 {
			if i != 0 {
				chunk++
			}

			currentChunkSize := chunkSize
			if len(things)-i < chunkSize {
				currentChunkSize = len(things) - i
			}
			chunked[chunk] = make([]batchChunk, currentChunkSize)
		}
		chunked[chunk][i%chunkSize] = batchChunk{things[i], uuids[i]}
	}

	for _, chunk := range chunked {
		k := kind.THING_KIND

		q := gremlin.New().Raw("g")

		for _, thing := range chunk {
			q = q.Raw("\n")
			className := schema.AssertValidClassName(thing.thing.AtClass)
			vertexLabel := j.state.GetMappedClassName(className)
			sourceClassAlias := "classToBeAdded"

			q = q.AddV(string(vertexLabel)).
				As(sourceClassAlias).
				StringProperty(PROP_KIND, k.Name()).
				StringProperty(PROP_UUID, thing.uuid.String()).
				StringProperty(PROP_CLASS_ID, string(vertexLabel)).
				StringProperty(PROP_AT_CONTEXT, thing.thing.AtContext).
				Int64Property(PROP_CREATION_TIME_UNIX, thing.thing.CreationTimeUnix).
				Int64Property(PROP_LAST_UPDATE_TIME_UNIX, thing.thing.LastUpdateTimeUnix)

			var err error
			q, err = j.addEdgesToQuery(q, k, className, thing.thing.Schema, sourceClassAlias)
			if err != nil {
				return err
			}
		}

		_, err := j.client.Execute(q)
		if err != nil {
			return err
		}
	}

	return nil
}
