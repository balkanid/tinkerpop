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
	"sync"
	"testing"
	"time"
)

func newTestContainer() *synchronizedMap {
	return &synchronizedMap{internalMap: map[string]ResultSet{}, syncLock: sync.Mutex{}}
}

func evalRequest(requestID, gremlin string) *request {
	return &request{
		op:   stringOp,
		args: map[string]interface{}{"gremlin": gremlin},
	}
}

// TestReportIfSlowFiresWhenOverThreshold verifies a slow execution is reported
// with the rendered query once the result set completes.
func TestReportIfSlowFiresWhenOverThreshold(t *testing.T) {
	var mu sync.Mutex
	var got *SlowQueryInfo
	sq := &slowQueryConfig{
		threshold:       time.Millisecond,
		traversalSource: "g",
		reporter: func(info SlowQueryInfo) {
			mu.Lock()
			defer mu.Unlock()
			got = &info
		},
	}

	container := newTestContainer()
	rs := newChannelResultSetWithSlowQuery("req-slow", container, evalRequest("req-slow", "g.V().count()"), sq).(*channelResultSet)
	container.store("req-slow", rs)

	// Force the elapsed time over the threshold deterministically.
	rs.startTime = time.Now().Add(-10 * time.Millisecond)
	rs.Close()

	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("expected slow-query reporter to fire, but it did not")
	}
	if got.Query != "g.V().count()" {
		t.Errorf("unexpected query: got %q", got.Query)
	}
	if got.Op != stringOp {
		t.Errorf("unexpected op: got %q want %q", got.Op, stringOp)
	}
	if got.Duration < got.Threshold {
		t.Errorf("duration %s should be >= threshold %s", got.Duration, got.Threshold)
	}
}

// TestReportIfSlowSkipsFastQuery verifies fast executions are not reported.
func TestReportIfSlowSkipsFastQuery(t *testing.T) {
	fired := false
	sq := &slowQueryConfig{
		threshold:       time.Hour, // nothing in a test will exceed this
		traversalSource: "g",
		reporter:        func(SlowQueryInfo) { fired = true },
	}

	container := newTestContainer()
	rs := newChannelResultSetWithSlowQuery("req-fast", container, evalRequest("req-fast", "g.V()"), sq).(*channelResultSet)
	container.store("req-fast", rs)
	rs.Close()

	if fired {
		t.Error("reporter fired for a fast query")
	}
}

// TestSlowQueryDisabledByDefault verifies that with no config the result set
// behaves exactly as before: no start time recorded, no reporting.
func TestSlowQueryDisabledByDefault(t *testing.T) {
	container := newTestContainer()
	rs := newChannelResultSetWithSlowQuery("req-off", container, evalRequest("req-off", "g.V()"), nil).(*channelResultSet)
	container.store("req-off", rs)

	if !rs.startTime.IsZero() {
		t.Error("startTime should be zero when slow-query logging is disabled")
	}
	rs.Close() // must not panic with a nil config
}

// TestExtractWritePartition verifies the tenant is read from the
// PartitionStrategy embedded in the bytecode.
func TestExtractWritePartition(t *testing.T) {
	bc := &Bytecode{
		sourceInstructions: []instruction{
			{
				operator: "withStrategies",
				arguments: []interface{}{
					&traversalStrategy{
						name:          decorationNamespace + "PartitionStrategy",
						configuration: map[string]interface{}{"writePartition": "tenant-42"},
					},
				},
			},
		},
	}
	if got := extractWritePartition(bc); got != "tenant-42" {
		t.Errorf("extractWritePartition = %q, want %q", got, "tenant-42")
	}

	// No partition strategy present -> empty string.
	if got := extractWritePartition(&Bytecode{}); got != "" {
		t.Errorf("extractWritePartition on empty bytecode = %q, want empty", got)
	}
}
