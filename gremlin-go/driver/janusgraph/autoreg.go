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

// Package janusgraph provides JanusGraph-specific types and codecs for gremlin-go.
//
// To use JanusGraph types, import this package with the blank identifier:
//
//	import _ "github.com/apache/tinkerpop/gremlin-go/v3/driver/janusgraph"
//
// This will automatically register the JanusGraph codecs for serialization
// and deserialization with the gremlin-go driver.
package janusgraph

import (
	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"reflect"
)

func init() {
	// Auto-register JanusGraph codecs when package is imported
	gremlingo.RegisterCustomTypeCodec(&janusGraphPCodec{})
	gremlingo.RegisterCustomTypeCodec(&relationIdentifierCodec{})

	// Register translators
	gremlingo.RegisterCustomTranslator(reflect.TypeOf(janusGraphP{}), translateJanusGraphPredicate)
	gremlingo.RegisterCustomTranslator(reflect.TypeOf(&janusGraphP{}), translateJanusGraphPredicate)
}
