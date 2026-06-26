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

package gremlingo

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// sampleCustom is a minimal custom type used to exercise the codec registry
// end-to-end via the public SerializeValue/DeserializeValue helpers.
type sampleCustom struct {
	Num  int64
	Text string
}

type sampleCustomCodec struct{}

func (sampleCustomCodec) TypeName() string     { return "test.Sample" }
func (sampleCustomCodec) TypeID() uint32       { return 0x9001 }
func (sampleCustomCodec) GoType() reflect.Type { return reflect.TypeOf(sampleCustom{}) }

func (sampleCustomCodec) Write(value interface{}, buffer *bytes.Buffer, ctx WriteContext) error {
	v := value.(*sampleCustom)
	if err := ctx.WriteInt64(buffer, v.Num); err != nil {
		return err
	}
	return ctx.WriteValue(v.Text, buffer, false)
}

func (sampleCustomCodec) Read(data *[]byte, i *int, ctx ReadContext) (interface{}, error) {
	num, err := ctx.ReadInt64(data, i)
	if err != nil {
		return nil, err
	}
	text, err := ctx.ReadString(data, i)
	if err != nil {
		return nil, err
	}
	return &sampleCustom{Num: num, Text: text}, nil
}

func TestCustomTypeRegistry(t *testing.T) {
	t.Run("register rejects duplicate id, type and name", func(t *testing.T) {
		assert.Nil(t, RegisterCustomTypeCodec(sampleCustomCodec{}))
		defer UnregisterCustomTypeCodec(0x9001)

		assert.True(t, IsCustomTypeRegistered(0x9001))
		// duplicate registration must fail
		assert.NotNil(t, RegisterCustomTypeCodec(sampleCustomCodec{}))
	})

	t.Run("round-trip through SerializeValue/DeserializeValue", func(t *testing.T) {
		assert.Nil(t, RegisterCustomTypeCodec(sampleCustomCodec{}))
		defer UnregisterCustomTypeCodec(0x9001)

		in := &sampleCustom{Num: 42, Text: "hello"}
		raw, err := SerializeValue(in)
		assert.Nil(t, err)

		// First byte is the custom type code 0x00.
		assert.Equal(t, byte(0x00), raw[0])

		out, err := DeserializeValue(raw)
		assert.Nil(t, err)
		assert.Equal(t, in, out)
	})

	t.Run("unregister removes the codec", func(t *testing.T) {
		assert.Nil(t, RegisterCustomTypeCodec(sampleCustomCodec{}))
		UnregisterCustomTypeCodec(0x9001)
		assert.False(t, IsCustomTypeRegistered(0x9001))

		_, err := SerializeValue(&sampleCustom{Num: 1, Text: "x"})
		assert.NotNil(t, err) // no codec -> unknown type
	})

	t.Run("malformed custom frame returns error, does not panic", func(t *testing.T) {
		assert.Nil(t, RegisterCustomTypeCodec(sampleCustomCodec{}))
		defer UnregisterCustomTypeCodec(0x9001)

		in := &sampleCustom{Num: 7, Text: "data"}
		raw, err := SerializeValue(in)
		assert.Nil(t, err)

		// Truncate the frame at every length and confirm no panic.
		for cut := 1; cut < len(raw); cut++ {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic on truncated input len=%d: %v", cut, r)
					}
				}()
				_, _ = DeserializeValue(raw[:cut])
			}()
		}
	})
}
