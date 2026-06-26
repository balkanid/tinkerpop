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
	"reflect"
	"sync"
)

// CustomTypeCodec defines the interface for custom GraphBinary type serialization.
// Types implementing this interface can be registered to handle custom type codes
// beyond the standard Gremlin types.
type CustomTypeCodec interface {
	// TypeName returns the fully qualified type name (e.g., "janusgraph.P")
	TypeName() string

	// TypeID returns the unique type identifier (e.g., 0x1002)
	TypeID() uint32

	// GoType returns the reflect.Type that this codec handles
	GoType() reflect.Type

	// Write serializes the value to GraphBinary format.
	// The buffer already has the type name, type ID, and second custom type code written.
	// The codec should write only the value-specific payload.
	Write(value interface{}, buffer *bytes.Buffer, ctx WriteContext) error

	// Read deserializes a value from GraphBinary format.
	// The buffer position is at the start of the value-specific payload
	// (after type name, type ID, and second custom type code).
	Read(data *[]byte, i *int, ctx ReadContext) (interface{}, error)
}

// WriteContext provides helper methods for writing GraphBinary values.
type WriteContext interface {
	// Write serializes a value in fully-qualified format
	Write(value interface{}, buffer *bytes.Buffer) error

	// WriteValue serializes a value without type information
	WriteValue(value interface{}, buffer *bytes.Buffer, nullable bool) error

	// WriteInt32 writes an int32 in big-endian format
	WriteInt32(buffer *bytes.Buffer, value int32) error

	// WriteInt64 writes an int64 in big-endian format
	WriteInt64(buffer *bytes.Buffer, value int64) error

	// WriteUInt32 writes a uint32 in big-endian format
	WriteUInt32(buffer *bytes.Buffer, value uint32) error

	// WriteString writes a GraphBinary string
	WriteString(buffer *bytes.Buffer, value string) error
}

// ReadContext provides helper methods for reading GraphBinary values.
type ReadContext interface {
	// ReadFullyQualifiedNullable reads a fully-qualified nullable value
	ReadFullyQualifiedNullable(data *[]byte, i *int, nullable bool) (interface{}, error)

	// ReadInt32 reads an int32 in big-endian format
	ReadInt32(data *[]byte, i *int) (int32, error)

	// ReadInt64 reads an int64 in big-endian format
	ReadInt64(data *[]byte, i *int) (int64, error)

	// ReadUInt32 reads a uint32 in big-endian format
	ReadUInt32(data *[]byte, i *int) (uint32, error)

	// ReadString reads a GraphBinary string
	ReadString(data *[]byte, i *int) (string, error)

	// ReadByteValue reads a single byte
	ReadByteValue(data *[]byte, i *int) (byte, error)

	// ReadBytes reads n bytes
	ReadBytes(data *[]byte, i *int, n int) ([]byte, error)
}

// customTypeWriteContext implements WriteContext
type customTypeWriteContext struct {
	serializer *graphBinaryTypeSerializer
}

func (ctx *customTypeWriteContext) Write(value interface{}, buffer *bytes.Buffer) error {
	_, err := ctx.serializer.write(value, buffer)
	return err
}

func (ctx *customTypeWriteContext) WriteValue(value interface{}, buffer *bytes.Buffer, nullable bool) error {
	_, err := ctx.serializer.writeValue(value, buffer, nullable)
	return err
}

func (ctx *customTypeWriteContext) WriteInt32(buffer *bytes.Buffer, value int32) error {
	return binary.Write(buffer, binary.BigEndian, value)
}

func (ctx *customTypeWriteContext) WriteInt64(buffer *bytes.Buffer, value int64) error {
	return binary.Write(buffer, binary.BigEndian, value)
}

func (ctx *customTypeWriteContext) WriteUInt32(buffer *bytes.Buffer, value uint32) error {
	return binary.Write(buffer, binary.BigEndian, value)
}

func (ctx *customTypeWriteContext) WriteString(buffer *bytes.Buffer, value string) error {
	err := binary.Write(buffer, binary.BigEndian, int32(len(value)))
	if err != nil {
		return err
	}
	buffer.WriteString(value)
	return nil
}

// customTypeReadContext implements ReadContext. Reads use package-level helpers,
// so no serializer reference is needed.
type customTypeReadContext struct{}

func (ctx *customTypeReadContext) ReadFullyQualifiedNullable(data *[]byte, i *int, nullable bool) (interface{}, error) {
	return readFullyQualifiedNullable(data, i, nullable)
}

func (ctx *customTypeReadContext) ReadInt32(data *[]byte, i *int) (int32, error) {
	if *i+4 > len(*data) {
		return 0, fmt.Errorf("not enough bytes to read int32: need 4, have %d", len(*data)-*i)
	}
	return readIntSafe(data, i), nil
}

func (ctx *customTypeReadContext) ReadInt64(data *[]byte, i *int) (int64, error) {
	if *i+8 > len(*data) {
		return 0, fmt.Errorf("not enough bytes to read int64: need 8, have %d", len(*data)-*i)
	}
	return readLongSafe(data, i), nil
}

func (ctx *customTypeReadContext) ReadUInt32(data *[]byte, i *int) (uint32, error) {
	if *i+4 > len(*data) {
		return 0, fmt.Errorf("not enough bytes to read uint32: need 4, have %d", len(*data)-*i)
	}
	v := binary.BigEndian.Uint32((*data)[*i : *i+4])
	*i += 4
	return v, nil
}

func (ctx *customTypeReadContext) ReadString(data *[]byte, i *int) (string, error) {
	if *i+4 > len(*data) {
		return "", fmt.Errorf("not enough bytes to read string length")
	}
	sz := int(readUint32Safe(data, i))
	if sz < 0 || *i+sz > len(*data) {
		return "", fmt.Errorf("invalid string length %d", sz)
	}
	s := string((*data)[*i : *i+sz])
	*i += sz
	return s, nil
}

func (ctx *customTypeReadContext) ReadByteValue(data *[]byte, i *int) (byte, error) {
	if *i+1 > len(*data) {
		return 0, fmt.Errorf("not enough bytes to read byte")
	}
	v := readByteSafe(data, i)
	return v, nil
}

func (ctx *customTypeReadContext) ReadBytes(data *[]byte, i *int, n int) ([]byte, error) {
	if *i+n > len(*data) {
		return nil, fmt.Errorf("not enough bytes: need %d, have %d", n, len(*data)-*i)
	}
	result := make([]byte, n)
	copy(result, (*data)[*i:*i+n])
	*i += n
	return result, nil
}

// CustomTypeRegistry manages registered custom type codecs.
type CustomTypeRegistry struct {
	mu          sync.RWMutex
	byType      map[reflect.Type]CustomTypeCodec
	byID        map[uint32]CustomTypeCodec
	byName      map[string]CustomTypeCodec
	translators map[reflect.Type]func(interface{}) (string, bool)
}

// globalCustomTypeRegistry is the default global registry
var globalCustomTypeRegistry = &CustomTypeRegistry{
	byType:      make(map[reflect.Type]CustomTypeCodec),
	byID:        make(map[uint32]CustomTypeCodec),
	byName:      make(map[string]CustomTypeCodec),
	translators: make(map[reflect.Type]func(interface{}) (string, bool)),
}

// RegisterCustomTypeCodec registers a custom type codec.
// Returns an error if:
// - The type ID is already registered
// - The Go type is already registered
// - The type name is already registered
func RegisterCustomTypeCodec(codec CustomTypeCodec) error {
	return globalCustomTypeRegistry.Register(codec)
}

// UnregisterCustomTypeCodec removes a custom type codec by type ID.
func UnregisterCustomTypeCodec(typeID uint32) {
	globalCustomTypeRegistry.Unregister(typeID)
}

// IsCustomTypeRegistered reports whether a codec is registered for the given
// type ID. Vendor packages can use this to make their registration idempotent.
func IsCustomTypeRegistered(typeID uint32) bool {
	return globalCustomTypeRegistry.GetCodecByID(typeID) != nil
}

// RegisterCustomTranslator registers a translation function for a custom type.
// The function should return the Gremlin string representation and true if it handled the type,
// or ("", false) if it cannot handle the value.
func RegisterCustomTranslator(goType reflect.Type, fn func(interface{}) (string, bool)) {
	globalCustomTypeRegistry.RegisterTranslator(goType, fn)
}

// UnregisterCustomTranslator removes a custom translator by Go type.
func UnregisterCustomTranslator(goType reflect.Type) {
	globalCustomTypeRegistry.UnregisterTranslator(goType)
}

// Register adds a codec to the registry.
func (r *CustomTypeRegistry) Register(codec CustomTypeCodec) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	goType := codec.GoType()
	typeID := codec.TypeID()
	typeName := codec.TypeName()

	if existing, ok := r.byID[typeID]; ok {
		return fmt.Errorf("custom type ID 0x%x already registered for %s", typeID, existing.TypeName())
	}

	if existing, ok := r.byType[goType]; ok {
		return fmt.Errorf("Go type %s already registered for type ID 0x%x", goType, existing.TypeID())
	}

	if existing, ok := r.byName[typeName]; ok {
		return fmt.Errorf("type name %q already registered for type ID 0x%x", typeName, existing.TypeID())
	}

	r.byType[goType] = codec
	r.byID[typeID] = codec
	r.byName[typeName] = codec

	return nil
}

// Unregister removes a codec from the registry by type ID.
func (r *CustomTypeRegistry) Unregister(typeID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if codec, ok := r.byID[typeID]; ok {
		delete(r.byType, codec.GoType())
		delete(r.byID, typeID)
		delete(r.byName, codec.TypeName())
	}
}

// RegisterTranslator adds a translation function for a custom type.
func (r *CustomTypeRegistry) RegisterTranslator(goType reflect.Type, fn func(interface{}) (string, bool)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.translators[goType] = fn
}

// UnregisterTranslator removes a custom translator.
func (r *CustomTypeRegistry) UnregisterTranslator(goType reflect.Type) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.translators, goType)
}

// GetCodecByType returns the codec for a Go type, or nil if not found.
func (r *CustomTypeRegistry) GetCodecByType(goType reflect.Type) CustomTypeCodec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	return r.byType[goType]
}

// GetCodecByID returns the codec for a type ID, or nil if not found.
func (r *CustomTypeRegistry) GetCodecByID(typeID uint32) CustomTypeCodec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byID[typeID]
}

// GetCodecByName returns the codec for a type name, or nil if not found.
func (r *CustomTypeRegistry) GetCodecByName(typeName string) CustomTypeCodec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[typeName]
}

// GetTranslator returns the translation function for a Go type, or nil if not found.
func (r *CustomTypeRegistry) GetTranslator(goType reflect.Type) func(interface{}) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	return r.translators[goType]
}

// writeContextForSerializer creates a WriteContext for a graphBinaryTypeSerializer
func writeContextForSerializer(s *graphBinaryTypeSerializer) WriteContext {
	return &customTypeWriteContext{serializer: s}
}
