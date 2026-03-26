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
	"bytes"
	"fmt"
	"reflect"
	"strings"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
)

const (
	janusGraphPTypeName               = "janusgraph.P"
	janusGraphPTypeID          uint32 = 0x1002
	relationIdentifierTypeName        = "janusgraph.RelationIdentifier"
	relationIdentifierTypeID   uint32 = 0x1001
)

// JanusGraphPredicate provides JanusGraph-specific text predicates.
type JanusGraphPredicate interface {
	TextContains(value interface{}) JanusGraphPredicate
	TextNotContains(value interface{}) JanusGraphPredicate
	TextContainsPrefix(value interface{}) JanusGraphPredicate
	TextNotContainsPrefix(value interface{}) JanusGraphPredicate
	TextContainsRegex(value interface{}) JanusGraphPredicate
	TextNotContainsRegex(value interface{}) JanusGraphPredicate
	TextContainsFuzzy(value interface{}) JanusGraphPredicate
	TextNotContainsFuzzy(value interface{}) JanusGraphPredicate
	TextContainsPhrase(value interface{}) JanusGraphPredicate
	TextNotContainsPhrase(value interface{}) JanusGraphPredicate
	TextPrefix(value interface{}) JanusGraphPredicate
	TextNotPrefix(value interface{}) JanusGraphPredicate
	TextRegex(value interface{}) JanusGraphPredicate
	TextNotRegex(value interface{}) JanusGraphPredicate
	TextFuzzy(value interface{}) JanusGraphPredicate
	TextNotFuzzy(value interface{}) JanusGraphPredicate
}

type janusGraphP struct {
	operator string
	value    interface{}
}

var Text JanusGraphPredicate = &janusGraphP{}

func newJanusGraphP(operator string, value interface{}) JanusGraphPredicate {
	return &janusGraphP{operator: operator, value: value}
}

func (*janusGraphP) TextContains(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textContains", value)
}

func (*janusGraphP) TextNotContains(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotContains", value)
}

func (*janusGraphP) TextContainsPrefix(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textContainsPrefix", value)
}

func (*janusGraphP) TextNotContainsPrefix(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotContainsPrefix", value)
}

func (*janusGraphP) TextContainsRegex(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textContainsRegex", value)
}

func (*janusGraphP) TextNotContainsRegex(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotContainsRegex", value)
}

func (*janusGraphP) TextContainsFuzzy(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textContainsFuzzy", value)
}

func (*janusGraphP) TextNotContainsFuzzy(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotContainsFuzzy", value)
}

func (*janusGraphP) TextContainsPhrase(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textContainsPhrase", value)
}

func (*janusGraphP) TextNotContainsPhrase(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotContainsPhrase", value)
}

func (*janusGraphP) TextPrefix(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textPrefix", value)
}

func (*janusGraphP) TextNotPrefix(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotPrefix", value)
}

func (*janusGraphP) TextRegex(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textRegex", value)
}

func (*janusGraphP) TextNotRegex(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotRegex", value)
}

func (*janusGraphP) TextFuzzy(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textFuzzy", value)
}

func (*janusGraphP) TextNotFuzzy(value interface{}) JanusGraphPredicate {
	return newJanusGraphP("textNotFuzzy", value)
}

func (p *janusGraphP) GetOperator() string {
	return p.operator
}

func (p *janusGraphP) GetValue() interface{} {
	return p.value
}

const stringEncodingMarker = "S"

const baseSymbols = "0123456789abcdefghijklmnopqrstuvwxyz"

// RelationIdentifier represents a JanusGraph edge/vertex-property identifier.
type RelationIdentifier struct {
	OutVertexID interface{}
	TypeID      int64
	RelationID  int64
	InVertexID  interface{}
	stringRep   string
}

// String returns the string representation of the RelationIdentifier.
func (r *RelationIdentifier) String() string {
	if r.stringRep != "" {
		return r.stringRep
	}
	return r.stringRep
}

// NewRelationIdentifier creates a new RelationIdentifier.
func NewRelationIdentifier(outVertexID interface{}, typeID int64, relationID int64, inVertexID interface{}) *RelationIdentifier {
	ri := &RelationIdentifier{
		OutVertexID: outVertexID,
		TypeID:      typeID,
		RelationID:  relationID,
		InVertexID:  inVertexID,
	}
	ri.stringRep = ri.buildString()
	return ri
}

func (r *RelationIdentifier) buildString() string {
	parts := make([]string, 0, 4)
	parts = append(parts, longEncode(r.RelationID))
	parts = append(parts, "-")
	if vid, ok := r.OutVertexID.(int64); ok {
		parts = append(parts, longEncode(vid))
	} else {
		parts = append(parts, stringEncodingMarker)
		parts = append(parts, fmt.Sprintf("%v", r.OutVertexID))
	}
	parts = append(parts, "-")
	parts = append(parts, longEncode(r.TypeID))
	if r.InVertexID != nil {
		parts = append(parts, "-")
		if vid, ok := r.InVertexID.(int64); ok {
			parts = append(parts, longEncode(vid))
		} else {
			parts = append(parts, stringEncodingMarker)
			parts = append(parts, fmt.Sprintf("%v", r.InVertexID))
		}
	}
	return strings.Join(parts, "")
}

func longEncode(num int64) string {
	if num == 0 {
		return "0"
	}
	absNum := num
	if num < 0 {
		absNum = -num
	}
	var chars []byte
	for absNum > 0 {
		chars = append(chars, baseSymbols[absNum%int64(len(baseSymbols))])
		absNum /= int64(len(baseSymbols))
	}
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	result := string(chars)
	if num < 0 {
		result = "-" + result
	}
	return result
}

func longDecode(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var num int64 = 0
	for _, ch := range s {
		num *= int64(len(baseSymbols))
		pos := strings.IndexRune(baseSymbols, ch)
		if pos < 0 {
			return 0, fmt.Errorf("invalid character '%c' in encoded long", ch)
		}
		num += int64(pos)
	}
	if neg {
		num = -num
	}
	return num, nil
}

// ParseRelationIdentifier parses a string into a RelationIdentifier.
func ParseRelationIdentifier(s string) (*RelationIdentifier, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 3 && len(parts) != 4 {
		return nil, fmt.Errorf("invalid relation identifier format: %s", s)
	}
	relationID, err := longDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid relation ID: %v", err)
	}
	var outVertexID interface{}
	if strings.HasPrefix(parts[1], stringEncodingMarker) {
		outVertexID = parts[1][1:]
	} else {
		vid, err := longDecode(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid out vertex ID: %v", err)
		}
		outVertexID = vid
	}
	typeID, err := longDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid type ID: %v", err)
	}
	var inVertexID interface{}
	if len(parts) == 4 {
		if strings.HasPrefix(parts[3], stringEncodingMarker) {
			inVertexID = parts[3][1:]
		} else {
			vid, err := longDecode(parts[3])
			if err != nil {
				return nil, fmt.Errorf("invalid in vertex ID: %v", err)
			}
			inVertexID = vid
		}
	}
	return NewRelationIdentifier(outVertexID, typeID, relationID, inVertexID), nil
}

// Equal checks if two RelationIdentifiers are equal.
func (r *RelationIdentifier) Equal(other *RelationIdentifier) bool {
	if other == nil {
		return false
	}
	return r.stringRep == other.stringRep
}

// Codec implementations

type janusGraphPCodec struct{}

func (c *janusGraphPCodec) TypeName() string {
	return janusGraphPTypeName
}

func (c *janusGraphPCodec) TypeID() uint32 {
	return janusGraphPTypeID
}

func (c *janusGraphPCodec) GoType() reflect.Type {
	return reflect.TypeOf(janusGraphP{})
}

func (c *janusGraphPCodec) Write(value interface{}, buffer *bytes.Buffer, ctx gremlingo.WriteContext) error {
	var v janusGraphP
	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		v = *(value.(*janusGraphP))
	} else {
		v = value.(janusGraphP)
	}

	if err := ctx.WriteValue(v.operator, buffer, false); err != nil {
		return err
	}
	return ctx.Write(v.value, buffer)
}

func (c *janusGraphPCodec) Read(data *[]byte, i *int, ctx gremlingo.ReadContext) (interface{}, error) {
	predicateName, err := ctx.ReadString(data, i)
	if err != nil {
		return nil, err
	}
	value, err := ctx.ReadFullyQualifiedNullable(data, i, true)
	if err != nil {
		return nil, err
	}
	return &janusGraphP{operator: predicateName, value: value}, nil
}

type relationIdentifierCodec struct{}

func (c *relationIdentifierCodec) TypeName() string {
	return relationIdentifierTypeName
}

func (c *relationIdentifierCodec) TypeID() uint32 {
	return relationIdentifierTypeID
}

func (c *relationIdentifierCodec) GoType() reflect.Type {
	return reflect.TypeOf(RelationIdentifier{})
}

const longMarker uint8 = 0
const stringMarker uint8 = 1

func (c *relationIdentifierCodec) Write(value interface{}, buffer *bytes.Buffer, ctx gremlingo.WriteContext) error {
	var v RelationIdentifier
	if reflect.TypeOf(value).Kind() == reflect.Ptr {
		v = *(value.(*RelationIdentifier))
	} else {
		v = value.(RelationIdentifier)
	}

	// Write out vertex ID
	if vid, ok := v.OutVertexID.(int64); ok {
		buffer.WriteByte(longMarker)
		if err := ctx.WriteInt64(buffer, vid); err != nil {
			return err
		}
	} else {
		buffer.WriteByte(stringMarker)
		writeJanusGraphString(buffer, fmt.Sprintf("%v", v.OutVertexID))
	}

	// Write type ID
	if err := ctx.WriteInt64(buffer, v.TypeID); err != nil {
		return err
	}

	// Write relation ID
	if err := ctx.WriteInt64(buffer, v.RelationID); err != nil {
		return err
	}

	// Write in vertex ID
	if v.InVertexID == nil {
		buffer.WriteByte(longMarker)
		return ctx.WriteInt64(buffer, 0)
	} else if vid, ok := v.InVertexID.(int64); ok {
		buffer.WriteByte(longMarker)
		return ctx.WriteInt64(buffer, vid)
	} else {
		buffer.WriteByte(stringMarker)
		writeJanusGraphString(buffer, fmt.Sprintf("%v", v.InVertexID))
		return nil
	}
}

func (c *relationIdentifierCodec) Read(data *[]byte, i *int, ctx gremlingo.ReadContext) (interface{}, error) {
	// Read out vertex marker
	outMarker, err := ctx.ReadByte(data, i)
	if err != nil {
		return nil, err
	}

	var outVertexID interface{}
	if outMarker == stringMarker {
		outVertexID = readJanusGraphString(data, i)
	} else {
		vid, err := ctx.ReadInt64(data, i)
		if err != nil {
			return nil, err
		}
		outVertexID = vid
	}

	// Read type ID
	typeID, err := ctx.ReadInt64(data, i)
	if err != nil {
		return nil, err
	}

	// Read relation ID
	relationID, err := ctx.ReadInt64(data, i)
	if err != nil {
		return nil, err
	}

	// Read in vertex marker
	inMarker, err := ctx.ReadByte(data, i)
	if err != nil {
		return nil, err
	}

	var inVertexID interface{}
	if inMarker == stringMarker {
		inVertexID = readJanusGraphString(data, i)
	} else {
		inVid, err := ctx.ReadInt64(data, i)
		if err != nil {
			return nil, err
		}
		if inVid == 0 {
			inVertexID = nil
		} else {
			inVertexID = inVid
		}
	}

	return NewRelationIdentifier(outVertexID, typeID, relationID, inVertexID), nil
}

func writeJanusGraphString(buffer *bytes.Buffer, s string) {
	b := []byte(s)
	for i, ch := range b {
		if i == len(b)-1 {
			buffer.WriteByte(ch | 0x80)
		} else {
			buffer.WriteByte(ch)
		}
	}
}

func readJanusGraphString(data *[]byte, i *int) string {
	var result []byte
	for {
		if *i >= len(*data) {
			break
		}
		b := (*data)[*i]
		*i++
		result = append(result, b&0x7F)
		if b&0x80 != 0 {
			break
		}
	}
	return string(result)
}

// Register registers all JanusGraph custom type codecs and translators.
// This must be called before using JanusGraph types with gremlin-go.
func Register() error {
	if err := gremlingo.RegisterCustomTypeCodec(&janusGraphPCodec{}); err != nil {
		return fmt.Errorf("failed to register janusgraph.P codec: %w", err)
	}
	if err := gremlingo.RegisterCustomTypeCodec(&relationIdentifierCodec{}); err != nil {
		return fmt.Errorf("failed to register janusgraph.RelationIdentifier codec: %w", err)
	}

	// Register translators
	gremlingo.RegisterCustomTranslator(reflect.TypeOf(janusGraphP{}), translateJanusGraphPredicate)
	gremlingo.RegisterCustomTranslator(reflect.TypeOf(&janusGraphP{}), translateJanusGraphPredicate)

	return nil
}

// Unregister removes all JanusGraph custom type codecs and translators.
func Unregister() {
	gremlingo.UnregisterCustomTypeCodec(janusGraphPTypeID)
	gremlingo.UnregisterCustomTypeCodec(relationIdentifierTypeID)
	gremlingo.UnregisterCustomTranslator(reflect.TypeOf(janusGraphP{}))
	gremlingo.UnregisterCustomTranslator(reflect.TypeOf(&janusGraphP{}))
}

func translateJanusGraphPredicate(v interface{}) (string, bool) {
	var p *janusGraphP
	switch val := v.(type) {
	case janusGraphP:
		p = &val
	case *janusGraphP:
		p = val
	default:
		return "", false
	}

	if p.operator == "" {
		return "", false
	}

	// Simplified translation - just operator(value)
	// For full translation we'd need access to the translator's toString method
	// which handles nested predicates, etc.
	return fmt.Sprintf("%s(%v)", p.operator, p.value), true
}
