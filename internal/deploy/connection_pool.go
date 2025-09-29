// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package deploy

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ConnectionPool manages SSH connections for reuse across multiple operations.
// This significantly improves performance for bulk operations by avoiding
// the overhead of establishing new connections for each operation.
type ConnectionPool struct {
	connections map[string]*pooledConnection
	mu          sync.RWMutex
	ttl         time.Duration
	maxConns    int
	cleanup     *time.Ticker
	done        chan struct{}
}

// pooledConnection represents a connection in the pool with metadata.
type pooledConnection struct {
	*ssh.Client
	sftpClient *sftp.Client
	lastUsed   time.Time
	inUse      bool
	key        string // host:user combination for identification
}

// NewConnectionPool creates a new SSH connection pool with the specified TTL and max connections.
func NewConnectionPool(ttl time.Duration, maxConns int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*pooledConnection),
		ttl:         ttl,
		maxConns:    maxConns,
		cleanup:     time.NewTicker(ttl / 2), // Cleanup twice as often as TTL
		done:        make(chan struct{}),
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// GetOrCreate retrieves an existing connection from the pool or creates a new one.
// It returns a pooled connection that should be released after use.
func (p *ConnectionPool) GetOrCreate(host, user, privateKey string, isBootstrap bool) (*PooledDeployer, error) {
	key := fmt.Sprintf("%s:%s", host, user)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we have a valid connection in the pool
	if conn, exists := p.connections[key]; exists && !conn.inUse {
		// Check if connection is still alive
		if time.Since(conn.lastUsed) < p.ttl {
			// Test the connection
			if _, _, err := conn.Client.SendRequest("keepalive", false, nil); err == nil {
				conn.inUse = true
				conn.lastUsed = time.Now()
				return &PooledDeployer{
					client:     conn.Client,
					sftpClient: conn.sftpClient,
					pool:       p,
					key:        key,
				}, nil
			}
		}
		// Connection is stale or dead, remove it
		p.removeConnection(key, false) // false = already have lock
	}

	// Check if we've reached the connection limit
	if len(p.connections) >= p.maxConns {
		// Find and remove the oldest unused connection
		p.evictOldestConnection(false) // false = already have lock
	}

	// Create a new connection
	deployer, err := newDeployerInternal(host, user, privateKey, isBootstrap)
	if err != nil {
		return nil, err
	}

	// Add to pool
	p.connections[key] = &pooledConnection{
		Client:     deployer.client,
		sftpClient: deployer.sftp,
		lastUsed:   time.Now(),
		inUse:      true,
		key:        key,
	}

	return &PooledDeployer{
		client:     deployer.client,
		sftpClient: deployer.sftp,
		pool:       p,
		key:        key,
	}, nil
}

// Release marks a connection as available for reuse.
func (p *ConnectionPool) Release(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[key]; exists {
		conn.inUse = false
		conn.lastUsed = time.Now()
	}
}

// Close closes all connections in the pool and stops the cleanup goroutine.
func (p *ConnectionPool) Close() {
	close(p.done)
	p.cleanup.Stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	for key := range p.connections {
		p.removeConnection(key, false) // false = already have lock
	}
}

// removeConnection removes a connection from the pool and closes it.
// If holdingLock is false, this function will acquire the lock.
func (p *ConnectionPool) removeConnection(key string, holdingLock bool) {
	if !holdingLock {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	if conn, exists := p.connections[key]; exists {
		if conn.sftpClient != nil {
			conn.sftpClient.Close()
		}
		if conn.Client != nil {
			conn.Client.Close()
		}
		delete(p.connections, key)
	}
}

// evictOldestConnection removes the oldest unused connection from the pool.
// If holdingLock is false, this function will acquire the lock.
func (p *ConnectionPool) evictOldestConnection(holdingLock bool) {
	if !holdingLock {
		p.mu.Lock()
		defer p.mu.Unlock()
	}

	var oldestKey string
	var oldestTime time.Time

	for key, conn := range p.connections {
		if !conn.inUse && (oldestKey == "" || conn.lastUsed.Before(oldestTime)) {
			oldestKey = key
			oldestTime = conn.lastUsed
		}
	}

	if oldestKey != "" {
		p.removeConnection(oldestKey, true) // true = already holding lock
	}
}

// cleanupLoop runs periodically to remove stale connections.
func (p *ConnectionPool) cleanupLoop() {
	for {
		select {
		case <-p.cleanup.C:
			p.cleanupStaleConnections()
		case <-p.done:
			return
		}
	}
}

// cleanupStaleConnections removes connections that have exceeded the TTL.
func (p *ConnectionPool) cleanupStaleConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for key, conn := range p.connections {
		if !conn.inUse && now.Sub(conn.lastUsed) > p.ttl {
			toRemove = append(toRemove, key)
		}
	}

	for _, key := range toRemove {
		p.removeConnection(key, true) // true = already holding lock
	}
}

// Stats returns statistics about the connection pool.
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalConnections: len(p.connections),
		InUseConnections: 0,
		MaxConnections:   p.maxConns,
	}

	for _, conn := range p.connections {
		if conn.inUse {
			stats.InUseConnections++
		}
	}

	return stats
}

// PoolStats represents statistics about the connection pool.
type PoolStats struct {
	TotalConnections int
	InUseConnections int
	MaxConnections   int
}

// PooledDeployer wraps a Deployer with connection pool management.
type PooledDeployer struct {
	client     *ssh.Client
	sftpClient *sftp.Client
	pool       *ConnectionPool
	key        string
}

// DeployAuthorizedKeys deploys authorized_keys content to the remote host.
func (pd *PooledDeployer) DeployAuthorizedKeys(content string) error {
	// Create a temporary Deployer for the actual deployment logic
	deployer := &Deployer{
		client: pd.client,
		sftp:   pd.sftpClient,
	}
	return deployer.DeployAuthorizedKeys(content)
}

// Close releases the connection back to the pool instead of closing it.
func (pd *PooledDeployer) Close() {
	pd.pool.Release(pd.key)
}

// GetSSHClient returns the underlying SSH client for advanced operations.
func (pd *PooledDeployer) GetSSHClient() *ssh.Client {
	return pd.client
}

// GetSFTPClient returns the underlying SFTP client for file operations.
func (pd *PooledDeployer) GetSFTPClient() *sftp.Client {
	return pd.sftpClient
}

// Default pool instance for global use
var defaultPool *ConnectionPool

// init initializes the default connection pool.
func init() {
	// Default settings: 5 minute TTL, max 20 connections
	defaultPool = NewConnectionPool(5*time.Minute, 20)
}

// GetDefaultPool returns the default connection pool instance.
func GetDefaultPool() *ConnectionPool {
	return defaultPool
}

// NewPooledDeployer creates a new pooled deployer using the default pool.
func NewPooledDeployer(host, user, privateKey string) (*PooledDeployer, error) {
	return defaultPool.GetOrCreate(host, user, privateKey, false)
}

// NewPooledBootstrapDeployer creates a new pooled deployer for bootstrap operations.
func NewPooledBootstrapDeployer(host, user, privateKey string) (*PooledDeployer, error) {
	return defaultPool.GetOrCreate(host, user, privateKey, true)
}