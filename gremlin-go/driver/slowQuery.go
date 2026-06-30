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

import "time"

// SlowQueryInfo describes a single traversal or script execution that exceeded
// the configured slow-query threshold. It is passed to a SlowQueryReporter so
// the caller can decide how to record it (structured logs, metrics, etc.).
type SlowQueryInfo struct {
	// RequestID is the server request UUID for the execution.
	RequestID string
	// Op is the request type: "bytecode" for traversals, "eval" for scripts.
	Op string
	// Query is the rendered Gremlin. For bytecode it is translated lazily and
	// only when the query is slow, so translation cost is never on the fast path.
	Query string
	// Tenant is the PartitionStrategy writePartition, when present on the query.
	Tenant string
	// Duration is the measured wall-clock time from submit to completion.
	Duration time.Duration
	// Threshold is the configured threshold that was exceeded.
	Threshold time.Duration
	// Err is the execution error, if the query completed with one.
	Err error
}

// slowQueryConfig holds the per-connection slow-query logging configuration. It
// is derived once from the connection settings and shared by all result sets on
// the connection.
type slowQueryConfig struct {
	threshold       time.Duration
	reporter        func(SlowQueryInfo)
	traversalSource string
}

// enabled reports whether slow-query logging should run. A nil config, a
// non-positive threshold, or a missing reporter all disable it, restoring the
// driver's original behavior with zero added work.
func (c *slowQueryConfig) enabled() bool {
	return c != nil && c.threshold > 0 && c.reporter != nil
}

// renderQuery produces the human-readable query string and tenant for a request.
// It is only called on the slow path (over threshold), so the bytecode
// translation cost is never paid for fast queries.
func (c *slowQueryConfig) renderQuery(req *request) (query, tenant string) {
	if req == nil {
		return "", ""
	}
	g, ok := req.args["gremlin"]
	if !ok {
		return "", ""
	}
	switch v := g.(type) {
	case string:
		// String ("eval") request: the gremlin arg is the script itself.
		return v, ""
	case Bytecode:
		if s, err := NewTranslator(c.traversalSource).Translate(&v); err == nil {
			query = s
		}
		tenant = extractWritePartition(&v)
		return query, tenant
	default:
		return "", ""
	}
}

// extractWritePartition pulls the PartitionStrategy writePartition out of the
// bytecode source instructions, when present. In this codebase that value is
// the tenant id (see GremlinStore.QueryWithCachingOption).
func extractWritePartition(bc *Bytecode) string {
	for _, insn := range bc.sourceInstructions {
		if insn.operator != "withStrategies" {
			continue
		}
		for _, arg := range insn.arguments {
			strategy, ok := arg.(*traversalStrategy)
			if !ok || strategy.name != decorationNamespace+"PartitionStrategy" {
				continue
			}
			if wp, ok := strategy.configuration["writePartition"].(string); ok {
				return wp
			}
		}
	}
	return ""
}
