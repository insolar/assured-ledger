package pool

import (
	"context"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/metrics"
	"github.com/insolar/assured-ledger/ledger-core/network/transport"
	"github.com/insolar/assured-ledger/ledger-core/rms/legacyhost"
)

// ConnectionPool interface provides methods to manage pool of network connections
type ConnectionPool interface {
	GetConnection(ctx context.Context, host *legacyhost.Host) (io.ReadWriter, error)
	CloseConnection(ctx context.Context, host *legacyhost.Host)
	Reset()
}

// NewConnectionPool constructor creates new ConnectionPool
func NewConnectionPool(t transport.StreamTransport) ConnectionPool {
	return newConnectionPool(t)
}

type connectionPool struct {
	transport transport.StreamTransport

	entryHolder *entryHolder
}

func newConnectionPool(t transport.StreamTransport) *connectionPool {
	return &connectionPool{
		transport:   t,
		entryHolder: newEntryHolder(),
	}
}

// GetConnection returns connection from the pool, if connection isn't exist, it will be created
func (cp *connectionPool) GetConnection(ctx context.Context, host *legacyhost.Host) (io.ReadWriter, error) {
	e := cp.getOrCreateEntry(ctx, host)
	return e.open(ctx)
}

// CloseConnection closes connection to the host
func (cp *connectionPool) CloseConnection(ctx context.Context, host *legacyhost.Host) {
	logger := inslogger.FromContext(ctx)

	logger.Debugf("[ CloseConnection ] Delete entry for connection to %s from pool", host)
	if cp.entryHolder.delete(host) {
		metrics.NetworkConnections.Dec()
	}
}

func (cp *connectionPool) getOrCreateEntry(ctx context.Context, host *legacyhost.Host) *entry {
	e, ok := cp.entryHolder.get(host)

	if ok {
		return e
	}

	logger := inslogger.FromContext(ctx)
	logger.Debugf("[ getOrCreateEntry ] Failed to retrieve entry for connection to %s, creating it", host)

	e = newEntry(cp.transport, nil, host, cp.CloseConnection)

	cp.entryHolder.add(host, e)
	size := cp.entryHolder.size()
	logger.Debugf(
		"[ getOrCreateEntry ] Added entry for connection to %s. Current pool size: %d",
		host,
		size,
	)
	metrics.NetworkConnections.Inc()

	return e
}

// Reset closes and removes all connections from the pool
func (cp *connectionPool) Reset() {
	cp.entryHolder.iterate(func(entry *entry) {
		entry.close()
	})
	cp.entryHolder.clear()
	metrics.NetworkConnections.Set(float64(cp.entryHolder.size()))
}
