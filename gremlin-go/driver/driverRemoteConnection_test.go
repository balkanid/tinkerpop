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
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestDriverRemoteConnection builds a DriverRemoteConnection around an empty,
// never-dialed pool so Close/ForceClose semantics can be tested without a live server.
func newTestDriverRemoteConnection(disableClose bool) *DriverRemoteConnection {
	pool := &loadBalancingPool{
		logHandler:             logger,
		newConnectionThreshold: newConnectionThreshold,
		loadBalanceLock:        sync.Mutex{},
	}
	client := &Client{
		logHandler:  logger,
		connections: pool,
	}
	return &DriverRemoteConnection{client: client, disableClose: disableClose}
}

func TestAuthentication(t *testing.T) {

	t.Run("Test BasicAuthInfo.", func(t *testing.T) {
		header := BasicAuthInfo("Lyndon", "Bauto")
		assert.Nil(t, header.GetHeader())
		b, _, _ := header.GetBasicAuth()
		assert.True(t, b)
	})

	t.Run("Test GetHeader.", func(t *testing.T) {
		header := &AuthInfo{}
		assert.Nil(t, header.GetHeader())
		header = nil
		assert.Nil(t, header.GetHeader())
		httpHeader := http.Header{}
		header = &AuthInfo{Header: httpHeader}
		assert.Equal(t, httpHeader, header.GetHeader())
	})
}

func TestDriverRemoteConnectionClose(t *testing.T) {
	t.Run("Close closes a connection with DisableClose unset", func(t *testing.T) {
		driver := newTestDriverRemoteConnection(false)
		driver.Close()
		assert.True(t, driver.isClosed)
	})

	t.Run("Close is a no-op when DisableClose is set", func(t *testing.T) {
		driver := newTestDriverRemoteConnection(true)
		driver.Close()
		assert.False(t, driver.isClosed)
	})

	t.Run("ForceClose closes a connection even when DisableClose is set", func(t *testing.T) {
		driver := newTestDriverRemoteConnection(true)
		driver.ForceClose()
		assert.True(t, driver.isClosed)
	})

	t.Run("IsClosed is false until Close/ForceClose", func(t *testing.T) {
		driver := newTestDriverRemoteConnection(true)
		assert.False(t, driver.IsClosed())
		driver.Close()
		assert.False(t, driver.IsClosed()) // DisableClose makes Close a no-op
		driver.ForceClose()
		assert.True(t, driver.IsClosed())
	})
}
