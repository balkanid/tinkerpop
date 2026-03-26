/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package janusgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJanusGraphPredicate(t *testing.T) {
	p := Text.TextContains("foo")
	assert.NotNil(t, p)

	p2 := Text.TextFuzzy("foobar")
	assert.NotNil(t, p2)

	p3 := Text.TextRegex("^foo.*bar$")
	assert.NotNil(t, p3)

	p4 := Text.TextPrefix("foo")
	assert.NotNil(t, p4)

	p5 := Text.TextContainsPrefix("lov")
	assert.NotNil(t, p5)
}

func TestRelationIdentifier(t *testing.T) {
	t.Run("NewRelationIdentifier with long IDs", func(t *testing.T) {
		ri := NewRelationIdentifier(int64(4336), int64(25621), int64(6174), int64(8248))
		assert.NotNil(t, ri)
		assert.Equal(t, int64(4336), ri.OutVertexID)
		assert.Equal(t, int64(25621), ri.TypeID)
		assert.Equal(t, int64(6174), ri.RelationID)
		assert.Equal(t, int64(8248), ri.InVertexID)
	})

	t.Run("NewRelationIdentifier with nil inVertexID", func(t *testing.T) {
		ri := NewRelationIdentifier(int64(4336), int64(25621), int64(6174), nil)
		assert.NotNil(t, ri)
		assert.Equal(t, int64(4336), ri.OutVertexID)
		assert.Equal(t, int64(25621), ri.TypeID)
		assert.Equal(t, int64(6174), ri.RelationID)
		assert.Nil(t, ri.InVertexID)
	})

	t.Run("NewRelationIdentifier with string IDs", func(t *testing.T) {
		ri := NewRelationIdentifier("jupiter", int64(25621), int64(6174), "pluto")
		assert.NotNil(t, ri)
		assert.Equal(t, "jupiter", ri.OutVertexID)
		assert.Equal(t, int64(25621), ri.TypeID)
		assert.Equal(t, int64(6174), ri.RelationID)
		assert.Equal(t, "pluto", ri.InVertexID)
	})

	t.Run("ParseRelationIdentifier", func(t *testing.T) {
		ri := NewRelationIdentifier(int64(100), int64(50), int64(200), nil)
		str := ri.String()
		assert.NotEmpty(t, str)

		decodedRi, err := ParseRelationIdentifier(str)
		assert.Nil(t, err)
		assert.Equal(t, int64(100), decodedRi.OutVertexID)
		assert.Equal(t, int64(50), decodedRi.TypeID)
		assert.Equal(t, int64(200), decodedRi.RelationID)
		assert.Nil(t, decodedRi.InVertexID)
	})

	t.Run("RelationIdentifier Equal", func(t *testing.T) {
		ri1 := NewRelationIdentifier(int64(100), int64(50), int64(200), nil)
		ri2 := NewRelationIdentifier(int64(100), int64(50), int64(200), nil)
		assert.True(t, ri1.Equal(ri2))

		ri3 := NewRelationIdentifier(int64(100), int64(50), int64(200), int64(300))
		assert.False(t, ri1.Equal(ri3))
	})
}

func TestRegisterUnregister(t *testing.T) {
	// Note: The janusgraph package auto-registers codecs via init() in autoreg.go
	// So Register() called here should fail since codecs are already registered

	// Register should fail (duplicate) since init() already registered
	err := Register()
	assert.NotNil(t, err) // Already registered
	assert.Contains(t, err.Error(), "already registered")

	// Unregister should work
	Unregister()

	// Register again should work now
	err = Register()
	assert.Nil(t, err)

	// Clean up
	Unregister()
}
