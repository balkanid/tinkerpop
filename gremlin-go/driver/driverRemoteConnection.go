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
	"crypto/tls"
	"runtime"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/language"
)

// DriverRemoteConnectionSettings are used to configure the DriverRemoteConnection.
type DriverRemoteConnectionSettings struct {
	session string

	TraversalSource          string
	TransporterType          TransporterType
	LogVerbosity             LogVerbosity
	Logger                   Logger
	Language                 language.Tag
	AuthInfo                 AuthInfoProvider
	TlsConfig                *tls.Config
	KeepAliveInterval        time.Duration
	WriteDeadline            time.Duration
	ConnectionTimeout        time.Duration
	EnableCompression        bool
	EnableUserAgentOnConnect bool
	ReadBufferSize           int
	WriteBufferSize          int

	// Minimum amount of concurrent active traversals on a connection to trigger creation of a new connection
	NewConnectionThreshold int
	// Maximum number of concurrent connections. Default: number of runtime processors
	MaximumConcurrentConnections int
	// Initial amount of instantiated connections. Default: 1
	InitialConcurrentConnections int
	// MaxConnectionLifetime bounds how long a pooled connection may live before it becomes eligible for
	// retirement. Retirement only happens once the connection is also idle (no in-flight results) - a
	// connection past its lifetime with active results is drained, never force-closed mid-request. A
	// random jitter of +/-20% is applied per-connection to avoid a synchronized reconnect burst across a
	// pool whose connections were mostly created together. Default: 0, meaning connections never expire.
	MaxConnectionLifetime time.Duration

	// LogPoolExpiration turns on dedicated logging (at Info verbosity) for connection
	// pool lifetime events: a connection exceeding MaxConnectionLifetime being closed
	// or drained, and a new connection being created to replace one that expired.
	// Has no effect if MaxConnectionLifetime is 0. Default: false.
	LogPoolExpiration bool

	// SlowQueryThreshold enables slow-query logging when set to a positive
	// duration. Any traversal or script whose end-to-end execution time meets
	// or exceeds this threshold is passed to SlowQueryReporter. A zero value
	// (the default) disables slow-query logging entirely.
	SlowQueryThreshold time.Duration
	// SlowQueryReporter is invoked once per slow execution. It is required for
	// slow-query logging to run; if nil, logging is disabled. The reporter is
	// called off the result set's lock, but on the protocol read goroutine, so
	// it should be fast and non-blocking (e.g. emit a log line or metric).
	SlowQueryReporter func(SlowQueryInfo)
	// SlowQueryMaxLength caps the rendered query string passed to
	// SlowQueryReporter, in bytes. When the rendered query exceeds this length,
	// it is cut down and SlowQueryInfo.QueryTruncated is set, with
	// SlowQueryInfo.QueryLength giving the original length. A non-positive
	// value (the default) disables truncation, so large queries/mutations are
	// still fully logged unless this is set.
	SlowQueryMaxLength int

	// DisableClose makes Close a no-op on this DriverRemoteConnection. Intended for a
	// connection a caller shares across many independent request handlers, each of
	// which follows the usual create-connection/defer-Close lifecycle without knowing
	// the connection is shared - without this, the first caller to finish would tear
	// the connection down for everyone else. The connection can still be torn down
	// deliberately via ForceClose. Default: false.
	DisableClose bool
}

// DriverRemoteConnection is a remote connection.
type DriverRemoteConnection struct {
	client          *Client
	spawnedSessions []*DriverRemoteConnection
	isClosed        bool
	disableClose    bool
	settings        *DriverRemoteConnectionSettings
}

// NewDriverRemoteConnection creates a new DriverRemoteConnection.
// If no custom connection settings are passed in, a connection will be created with "g" as the default TraversalSource,
// Gorilla as the default Transporter, Info as the default LogVerbosity, a default logger struct, and English and as the
// default language
func NewDriverRemoteConnection(
	url string,
	configurations ...func(settings *DriverRemoteConnectionSettings)) (*DriverRemoteConnection, error) {
	settings := &DriverRemoteConnectionSettings{
		session: "",

		TraversalSource:          "g",
		TransporterType:          Gorilla,
		LogVerbosity:             Info,
		Logger:                   &defaultLogger{},
		Language:                 language.English,
		AuthInfo:                 &AuthInfo{},
		TlsConfig:                &tls.Config{},
		KeepAliveInterval:        keepAliveIntervalDefault,
		WriteDeadline:            writeDeadlineDefault,
		ConnectionTimeout:        connectionTimeoutDefault,
		EnableCompression:        false,
		EnableUserAgentOnConnect: true,
		// ReadBufferSize and WriteBufferSize specify I/O buffer sizes in bytes. The default is 1048576.
		// If a buffer size is set zero, then the Gorilla websocket 4096 default size is used. The I/O buffer
		// sizes do not limit the size of the messages that can be sent or received.
		ReadBufferSize:  1048576,
		WriteBufferSize: 1048576,

		NewConnectionThreshold:       defaultNewConnectionThreshold,
		MaximumConcurrentConnections: runtime.NumCPU(),
		InitialConcurrentConnections: defaultInitialConcurrentConnections,
		MaxConnectionLifetime:        0,
	}
	for _, configuration := range configurations {
		configuration(settings)
	}

	connSettings := &connectionSettings{
		authInfo:                 settings.AuthInfo,
		tlsConfig:                settings.TlsConfig,
		keepAliveInterval:        settings.KeepAliveInterval,
		writeDeadline:            settings.WriteDeadline,
		connectionTimeout:        settings.ConnectionTimeout,
		enableCompression:        settings.EnableCompression,
		readBufferSize:           settings.ReadBufferSize,
		writeBufferSize:          settings.WriteBufferSize,
		enableUserAgentOnConnect: settings.EnableUserAgentOnConnect,
		slowQueryThreshold:       settings.SlowQueryThreshold,
		slowQueryReporter:        settings.SlowQueryReporter,
		slowQueryMaxLength:       settings.SlowQueryMaxLength,
		traversalSource:          settings.TraversalSource,
		maxConnectionLifetime:    settings.MaxConnectionLifetime,
		logPoolExpiration:        settings.LogPoolExpiration,
	}

	logHandler := newLogHandler(settings.Logger, settings.LogVerbosity, settings.Language)
	if settings.session != "" {
		logHandler.log(Debug, sessionDetected)
		settings.MaximumConcurrentConnections = 1
	}

	if settings.InitialConcurrentConnections > settings.MaximumConcurrentConnections {
		logHandler.logf(Warning, poolInitialExceedsMaximum, settings.InitialConcurrentConnections,
			settings.MaximumConcurrentConnections, settings.MaximumConcurrentConnections)
		settings.InitialConcurrentConnections = settings.MaximumConcurrentConnections
	}
	pool, err := newLoadBalancingPool(url, logHandler, connSettings, settings.NewConnectionThreshold,
		settings.MaximumConcurrentConnections, settings.InitialConcurrentConnections)
	if err != nil {
		if err != nil {
			logHandler.logf(Error, logErrorGeneric, "NewDriverRemoteConnection", err.Error())
		}
		return nil, err
	}

	client := &Client{
		url:             url,
		traversalSource: settings.TraversalSource,
		logHandler:      logHandler,
		transporterType: settings.TransporterType,
		connections:     pool,
		session:         settings.session,
	}

	return &DriverRemoteConnection{client: client, isClosed: false, disableClose: settings.DisableClose, settings: settings}, nil
}

// Close closes the DriverRemoteConnection. A no-op if DisableClose was set on the
// settings this connection was created with - use ForceClose to close it anyway.
// Errors if any will be logged
func (driver *DriverRemoteConnection) Close() {
	if driver.disableClose {
		return
	}
	driver.forceClose()
}

// ForceClose closes the DriverRemoteConnection unconditionally, ignoring DisableClose.
// Intended for a connection's true owner (e.g. process shutdown) to tear it down even
// when it was shared with callers whose own Close calls are no-ops.
func (driver *DriverRemoteConnection) ForceClose() {
	driver.forceClose()
}

// IsClosed reports whether this DriverRemoteConnection has been closed via Close or
// ForceClose. Used by shared-connection owners to decide when to dial a replacement.
func (driver *DriverRemoteConnection) IsClosed() bool {
	return driver.isClosed
}

func (driver *DriverRemoteConnection) forceClose() {
	// If DriverRemoteConnection has spawnedSessions then they must be closed as well.
	if len(driver.spawnedSessions) > 0 {
		driver.client.logHandler.logf(Debug, closingSpawnedSessions, driver.client.url)
		for _, session := range driver.spawnedSessions {
			session.Close()
		}
		driver.spawnedSessions = driver.spawnedSessions[:0]
	}

	if driver.isSession() {
		driver.client.logHandler.logf(Info, closeSession, driver.client.url, driver.client.session)
	} else {
		driver.client.logHandler.logf(Info, closeDriverRemoteConnection, driver.client.url)
	}
	driver.client.Close()
	driver.isClosed = true
}

// SubmitWithOptions sends a string traversal to the server along with specified RequestOptions.
func (driver *DriverRemoteConnection) SubmitWithOptions(traversalString string, requestOptions RequestOptions) (ResultSet, error) {
	result, err := driver.client.SubmitWithOptions(traversalString, requestOptions)
	if err != nil {
		driver.client.logHandler.logf(Error, logErrorGeneric, "Driver.Submit()", err.Error())
	}
	return result, err
}

// Submit sends a string traversal to the server.
func (driver *DriverRemoteConnection) Submit(traversalString string) (ResultSet, error) {
	return driver.SubmitWithOptions(traversalString, *new(RequestOptions))
}

// submitBytecode sends a Bytecode traversal to the server.
func (driver *DriverRemoteConnection) submitBytecode(bytecode *Bytecode) (ResultSet, error) {
	if driver.isClosed {
		return nil, newError(err0203SubmitBytecodeToClosedConnectionError)
	}
	return driver.client.submitBytecode(bytecode)
}

func (driver *DriverRemoteConnection) isSession() bool {
	return driver.client.session != ""
}

// CreateSession generates a new session. sessionId stores the optional UUID param. It can be used to create a session with a specific UUID.
func (driver *DriverRemoteConnection) CreateSession(sessionId ...string) (*DriverRemoteConnection, error) {
	if len(sessionId) > 1 {
		return nil, newError(err0201CreateSessionMultipleIdsError)
	} else if driver.isSession() {
		return nil, newError(err0202CreateSessionFromSessionError)
	}

	driver.client.logHandler.log(Info, creatingSessionConnection)
	drc, err := NewDriverRemoteConnection(driver.client.url, func(settings *DriverRemoteConnectionSettings) {
		if len(sessionId) == 1 {
			settings.session = sessionId[0]
		} else {
			settings.session = uuid.New().String()
		}
		// copy other settings from parent
		settings.TraversalSource = driver.settings.TraversalSource
		settings.TransporterType = driver.settings.TransporterType
		settings.Logger = driver.settings.Logger
		settings.LogVerbosity = driver.settings.LogVerbosity
		settings.Language = driver.settings.Language
		settings.AuthInfo = driver.settings.AuthInfo
		settings.TlsConfig = driver.settings.TlsConfig
		settings.KeepAliveInterval = driver.settings.KeepAliveInterval
		settings.WriteDeadline = driver.settings.WriteDeadline
		settings.ConnectionTimeout = driver.settings.ConnectionTimeout
		settings.NewConnectionThreshold = driver.settings.NewConnectionThreshold
		settings.MaxConnectionLifetime = driver.settings.MaxConnectionLifetime
		settings.LogPoolExpiration = driver.settings.LogPoolExpiration
		settings.EnableCompression = driver.settings.EnableCompression
		settings.ReadBufferSize = driver.settings.ReadBufferSize
		settings.WriteBufferSize = driver.settings.WriteBufferSize
		settings.MaximumConcurrentConnections = driver.settings.MaximumConcurrentConnections
		settings.SlowQueryThreshold = driver.settings.SlowQueryThreshold
		settings.SlowQueryReporter = driver.settings.SlowQueryReporter
		settings.SlowQueryMaxLength = driver.settings.SlowQueryMaxLength
	})
	if err != nil {
		return nil, err
	}
	driver.spawnedSessions = append(driver.spawnedSessions, drc)
	return drc, nil
}

func (driver *DriverRemoteConnection) GetSessionId() string {
	return driver.client.session
}

func (driver *DriverRemoteConnection) commit() (ResultSet, error) {
	bc := &Bytecode{}
	bc.AddSource("tx", "commit")
	return driver.submitBytecode(bc)
}

func (driver *DriverRemoteConnection) rollback() (ResultSet, error) {
	bc := &Bytecode{}
	bc.AddSource("tx", "rollback")
	return driver.submitBytecode(bc)
}
