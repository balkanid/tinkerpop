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
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
	"math/big"
	"reflect"
	"testing"
	"time"
)

func TestGraphBinaryV1(t *testing.T) {
	t.Run("graphBinaryTypeSerializer tests", func(t *testing.T) {
		serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}

		t.Run("getType should return correct type for simple type", func(t *testing.T) {
			res, err := serializer.getType(int32(1))
			assert.Nil(t, err)
			assert.Equal(t, intType, res)
		})
		t.Run("getType should return correct type for complex type", func(t *testing.T) {
			res, err := serializer.getType([]byte{1, 2, 3})
			assert.Nil(t, err)
			assert.Equal(t, listType, res)
		})
		t.Run("getType should return error for missing type", func(t *testing.T) {
			_, err := serializer.getType(Error)
			assert.NotNil(t, err)
		})

		t.Run("getWriter should return correct func for dataType", func(t *testing.T) {
			writer, err := serializer.getWriter(intType)
			assert.Nil(t, err)
			assert.Equal(t, reflect.ValueOf(intWriter).Pointer(), reflect.ValueOf(writer).Pointer())
		})
		t.Run("getWriter should return error for missing dataType", func(t *testing.T) {
			_, err := serializer.getWriter(nullType)
			assert.NotNil(t, err)
		})

		t.Run("getSerializerToWrite should return correct func for int", func(t *testing.T) {
			writer, dataType, err := serializer.getSerializerToWrite(int32(1))
			assert.Nil(t, err)
			assert.Equal(t, intType, dataType)
			assert.Equal(t, reflect.ValueOf(intWriter).Pointer(), reflect.ValueOf(writer).Pointer())
		})
		t.Run("getSerializerToWrite should return error for missing dataType", func(t *testing.T) {
			_, _, err := serializer.getSerializerToWrite(nullType)
			assert.NotNil(t, err)
		})
	})

	t.Run("read-write tests", func(t *testing.T) {
		t.Run("read-write string", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			str := "test string"
			buf, err := stringWriter(str, &buffer, nil)
			assert.Nil(t, err)
			res, err := readString(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, str, res)
		})
		t.Run("read-write GremlinType", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := &GremlinType{"test fqcn"}
			buf, err := classWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readClass(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write bool", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			f := func(value interface{}, buffer *bytes.Buffer, typeSerializer *graphBinaryTypeSerializer) ([]byte, error) {
				err := binary.Write(buffer, binary.BigEndian, value.(bool))
				return buffer.Bytes(), err
			}
			data, err := f(false, &buffer, nil)
			assert.Nil(t, err)
			res, err := readBoolean(&data, &pos)
			assert.Nil(t, err)
			assert.False(t, res.(bool))

			data, err = f(true, &buffer, nil)
			assert.Nil(t, err)
			res, err = readBoolean(&data, &pos)
			assert.Nil(t, err)
			assert.True(t, res.(bool))
		})
		t.Run("read-write BigDecimal", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := &BigDecimal{11, *big.NewInt(int64(22))}
			buf, err := bigDecimalWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readBigDecimal(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write int", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := int32(123)
			buf, err := intWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readInt(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write short", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := int16(123)
			buf, err := shortWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readShort(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write short int8", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := int8(123)
			buf, err := shortWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readShort(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, int16(source), res)
		})
		t.Run("read-write long", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := 123
			buf, err := longWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readLong(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, int64(source), res)
		})
		t.Run("read-write bigInt", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := big.NewInt(123)
			buf, err := bigIntWriter(*source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readBigInt(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write bigInt uint64", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := uint64(123)
			buf, err := bigIntWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readBigInt(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, new(big.Int).SetUint64(source), res)
		})
		t.Run("read-write bigInt uint64", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := uint(123)
			buf, err := bigIntWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readBigInt(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, new(big.Int).SetUint64(uint64(source)), res)
		})
		t.Run("read-write list", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := []interface{}{int32(111), "str"}
			buf, err := listWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readList(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write byteBuffer", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := &ByteBuffer{[]byte{byte(127), byte(255)}}
			buf, err := byteBufferWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readByteBuffer(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write set", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := NewSimpleSet(int32(111), "str")
			buf, err := setWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readSet(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write map", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := map[interface{}]interface{}{1: "s1", "s2": 2, nil: nil}
			buf, err := mapWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := readMap(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, fmt.Sprintf("%v", source), fmt.Sprintf("%v", res))
		})
		t.Run("read incomparable map: a map value as the key", func(t *testing.T) {
			// prepare test data
			var buf = &bytes.Buffer{}
			typeSerializer := &graphBinaryTypeSerializer{}
			// write the size of map
			err := binary.Write(buf, binary.BigEndian, uint32(1))
			if err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}
			// write a map value as the key
			k1 := map[string]string{"key": "value"}
			_, err = typeSerializer.write(reflect.ValueOf(k1).Interface(), buf)
			if err != nil {
				t.Fatalf("Failed to encode data: %v", err)
			}
			v1 := "value1"
			_, err = typeSerializer.write(reflect.ValueOf(v1).Interface(), buf)
			if err != nil {
				t.Fatalf("Failed to encode data: %v", err)
			}

			data := buf.Bytes()
			i := 0
			result, err := readMap(&data, &i)
			if err != nil {
				t.Fatalf("readMap failed: %v", err)
			}
			mResult, ok := result.(map[interface{}]interface{})
			if !ok {
				t.Fatalf("readMap result not map[interface{}]interface{}")
			}
			for k, v := range mResult {
				assert.Equal(t, reflect.Ptr, reflect.TypeOf(k).Kind())
				assert.Equal(t, "value1", v)
			}
		})
		t.Run("read incomparable map: a slice value as the key", func(t *testing.T) {
			// prepare test data
			var buf = &bytes.Buffer{}
			typeSerializer := &graphBinaryTypeSerializer{}
			// write the size of map
			err := binary.Write(buf, binary.BigEndian, uint32(1))
			if err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}
			// write a slice value as the key
			k2 := []int{1, 2, 3}
			_, err = typeSerializer.write(reflect.ValueOf(k2).Interface(), buf)
			if err != nil {
				t.Fatalf("Failed to encode data: %v", err)
			}
			v2 := "value2"
			_, err = typeSerializer.write(reflect.ValueOf(v2).Interface(), buf)
			if err != nil {
				t.Fatalf("Failed to encode data: %v", err)
			}

			data := buf.Bytes()
			i := 0
			result, err := readMap(&data, &i)
			if err != nil {
				t.Fatalf("readMap failed: %v", err)
			}
			expected := map[interface{}]interface{}{
				"[1 2 3]": "value2",
			}
			if !reflect.DeepEqual(result, expected) {
				t.Errorf("Expected %v, but got %v", expected, result)
			}
		})
		t.Run("read-write time", func(t *testing.T) {
			pos := 0
			var buffer bytes.Buffer
			source := time.Date(2022, 5, 10, 9, 51, 0, 0, time.Local)
			buf, err := timeWriter(source, &buffer, nil)
			assert.Nil(t, err)
			res, err := timeReader(&buf, &pos)
			assert.Nil(t, err)
			assert.Equal(t, source, res)
		})
		t.Run("read-write janusGraphP textContains", func(t *testing.T) {
			serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}
			var buffer bytes.Buffer
			source := JanusGraphText.TextContains("foo")

			_, err := serializer.write(source, &buffer)
			assert.Nil(t, err)
			encoded := buffer.Bytes()
			assert.GreaterOrEqual(t, len(encoded), 2)
			// Custom type code is 0x00
			assert.Equal(t, byte(customType), encoded[0])
			// Verify type name follows
			typeName := "janusgraph.P"
			typeInfoLen := int32(len(typeName))
			// Read length from bytes 1-4 (big endian)
			readLen := int32(encoded[1])<<24 | int32(encoded[2])<<16 | int32(encoded[3])<<8 | int32(encoded[4])
			assert.Equal(t, typeInfoLen, readLen)
			// Read type name string
			readTypeName := string(encoded[5 : 5+int(readLen)])
			assert.Equal(t, typeName, readTypeName)

			// Test round-trip: read back what we wrote
			i := 0
			result, err := readFullyQualifiedNullable(&encoded, &i, true)
			assert.Nil(t, err)
			assert.NotNil(t, result)
			jgp, ok := result.(*janusGraphP)
			assert.True(t, ok)
			assert.Equal(t, "textContains", jgp.operator)
			assert.Equal(t, "foo", jgp.value.(string))
		})
		t.Run("read-write janusGraphP textFuzzy", func(t *testing.T) {
			serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}
			var buffer bytes.Buffer
			source := JanusGraphText.TextFuzzy("foobar")

			_, err := serializer.write(source, &buffer)
			assert.Nil(t, err)
			encoded := buffer.Bytes()
			assert.GreaterOrEqual(t, len(encoded), 2)
		})
		t.Run("read-write janusGraphP textRegex", func(t *testing.T) {
			serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}
			var buffer bytes.Buffer
			source := JanusGraphText.TextRegex("^foo.*bar$")

			_, err := serializer.write(source, &buffer)
			assert.Nil(t, err)
			encoded := buffer.Bytes()
			assert.GreaterOrEqual(t, len(encoded), 2)
		})
		t.Run("read-write RelationIdentifier with long IDs", func(t *testing.T) {
			serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}
			var buffer bytes.Buffer
			source := NewRelationIdentifier(int64(4336), int64(25621), int64(6174), int64(8248))

			_, err := serializer.write(source, &buffer)
			assert.Nil(t, err)
			encoded := buffer.Bytes()
			assert.GreaterOrEqual(t, len(encoded), 2)

			// Verify round-trip
			i := 0
			result, err := readFullyQualifiedNullable(&encoded, &i, true)
			assert.Nil(t, err)
			assert.NotNil(t, result)
			ri, ok := result.(*RelationIdentifier)
			assert.True(t, ok)
			assert.Equal(t, int64(4336), ri.OutVertexID)
			assert.Equal(t, int64(25621), ri.TypeID)
			assert.Equal(t, int64(6174), ri.RelationID)
			assert.Equal(t, int64(8248), ri.InVertexID)
		})
		t.Run("read-write RelationIdentifier with nil inVertexID", func(t *testing.T) {
			serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}
			var buffer bytes.Buffer
			source := NewRelationIdentifier(int64(4336), int64(25621), int64(6174), nil)

			_, err := serializer.write(source, &buffer)
			assert.Nil(t, err)
			encoded := buffer.Bytes()

			i := 0
			result, err := readFullyQualifiedNullable(&encoded, &i, true)
			assert.Nil(t, err)
			ri, ok := result.(*RelationIdentifier)
			assert.True(t, ok)
			assert.Equal(t, int64(4336), ri.OutVertexID)
			assert.Equal(t, int64(25621), ri.TypeID)
			assert.Equal(t, int64(6174), ri.RelationID)
			assert.Nil(t, ri.InVertexID)
		})
		t.Run("read-write RelationIdentifier with string IDs", func(t *testing.T) {
			serializer := graphBinaryTypeSerializer{newLogHandler(&defaultLogger{}, Error, language.English)}
			var buffer bytes.Buffer
			source := NewRelationIdentifier("jupiter", int64(25621), int64(6174), "pluto")

			_, err := serializer.write(source, &buffer)
			assert.Nil(t, err)
			encoded := buffer.Bytes()

			i := 0
			result, err := readFullyQualifiedNullable(&encoded, &i, true)
			assert.Nil(t, err)
			ri, ok := result.(*RelationIdentifier)
			assert.True(t, ok)
			assert.Equal(t, "jupiter", ri.OutVertexID)
			assert.Equal(t, int64(25621), ri.TypeID)
			assert.Equal(t, int64(6174), ri.RelationID)
			assert.Equal(t, "pluto", ri.InVertexID)
		})
		t.Run("parse RelationIdentifier from string", func(t *testing.T) {
			// Test encoding/decoding of string format
			// Create a RelationIdentifier and verify its string representation
			// String format is: relationID-outVertexID-typeID[-inVertexID]
			ri := NewRelationIdentifier(int64(100), int64(50), int64(200), nil)
			str := ri.String()
			assert.NotEmpty(t, str)

			// Parse it back
			decodedRi, err := ParseRelationIdentifier(str)
			assert.Nil(t, err)
			// Fields: OutVertexID=100, TypeID=50, RelationID=200
			assert.Equal(t, int64(100), decodedRi.OutVertexID)
			assert.Equal(t, int64(50), decodedRi.TypeID)
			assert.Equal(t, int64(200), decodedRi.RelationID)
			assert.Nil(t, decodedRi.InVertexID)
		})
		t.Run("RelationIdentifier string representation", func(t *testing.T) {
			ri := NewRelationIdentifier(int64(4336), int64(25621), int64(6174), nil)
			str := ri.String()
			assert.Contains(t, str, "-")
		})
	})

	t.Run("error handle tests", func(t *testing.T) {
		t.Run("test map key not string failure", func(t *testing.T) {
			i := 0
			buff := []byte{0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x00, 0x00, 0x01}
			m, err := readMapUnqualified(&buff, &i)
			assert.Nil(t, m)
			assert.Equal(t, newError(err0703ReadMapNonStringKeyError, intType), err)
		})
	})
}
