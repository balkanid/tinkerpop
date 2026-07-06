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
	"fmt"
	"time"
	"unicode/utf8"
)

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
	// It is capped at the configured SlowQueryMaxLength; see QueryTruncated.
	Query string
	// QueryTruncated is true when Query was cut down from its original length
	// because it exceeded the configured SlowQueryMaxLength.
	QueryTruncated bool
	// QueryLength is the byte length of the rendered query before truncation.
	QueryLength int
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
	// maxQueryLength caps the rendered query string passed to the reporter, in
	// bytes. A non-positive value disables truncation (the default), so
	// existing callers are unaffected until they opt in.
	maxQueryLength int
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

// truncate caps query at maxQueryLength bytes, trimming back further if needed
// to avoid splitting a multi-byte UTF-8 rune at the cut point. It reports
// whether truncation occurred and the original (pre-truncation) byte length,
// so callers can log both the shortened query and how much was cut.
func (c *slowQueryConfig) truncate(query string) (result string, truncated bool, originalLength int) {
	originalLength = len(query)
	if c.maxQueryLength <= 0 || originalLength <= c.maxQueryLength {
		return query, false, originalLength
	}
	cut := query[:c.maxQueryLength]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return fmt.Sprintf("%s...[truncated, %d bytes total]", cut, originalLength), true, originalLength
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
