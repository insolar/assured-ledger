package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
)

var (
	udpMutex         sync.RWMutex
	datagramHandlers = make(map[string]DatagramHandler)
	tcpMutex         sync.RWMutex
	streamHandlers   = make(map[string]StreamHandler)
	listenersCount   = int32(0)
)

// NewFakeFactory constructor creates new fake transport factory
func NewFakeFactory(cfg configuration.Transport) Factory {
	return &fakeFactory{cfg: cfg}
}

type fakeFactory struct {
	cfg configuration.Transport
}

// CreateStreamTransport creates fake StreamTransport for tests
func (f *fakeFactory) CreateStreamTransport(handler StreamHandler) (StreamTransport, error) {
	return &fakeStreamTransport{address: f.cfg.Address, handler: handler}, nil
}

// CreateDatagramTransport creates fake DatagramTransport for tests
func (f *fakeFactory) CreateDatagramTransport(handler DatagramHandler) (DatagramTransport, error) {
	return &fakeDatagramTransport{address: f.cfg.Address, handler: handler}, nil
}

type fakeDatagramTransport struct {
	address string
	handler DatagramHandler
}

func (f *fakeDatagramTransport) Start(ctx context.Context) error {
	udpMutex.Lock()
	defer udpMutex.Unlock()

	datagramHandlers[f.address] = f.handler
	return nil
}

func (f *fakeDatagramTransport) Stop(ctx context.Context) error {
	udpMutex.Lock()
	defer udpMutex.Unlock()

	datagramHandlers[f.address] = nil
	return nil
}

func (f *fakeDatagramTransport) SendDatagram(ctx context.Context, address string, data []byte) error {
	global.Debugf("fakeDatagramTransport SendDatagram to %s : %v", address, data)

	if len(data) > udpMaxPacketSize {
		return errors.New(fmt.Sprintf("udpTransport.send: too big input data. Maximum: %d. Current: %d",
			udpMaxPacketSize, len(data)))
	}
	_, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return errors.W(err, "Failed to resolve UDP address")
	}

	udpMutex.RLock()
	defer udpMutex.RUnlock()

	h := datagramHandlers[address]
	if h != nil {
		cpData := make([]byte, len(data))
		copy(cpData, data)
		go h.HandleDatagram(ctx, f.address, cpData)
	}

	return nil
}

func (f *fakeDatagramTransport) Address() string {
	return f.address
}

type fakeStreamTransport struct {
	address string
	handler StreamHandler
	cancel  context.CancelFunc
	ctx     context.Context
}

func (f *fakeStreamTransport) Start(ctx context.Context) error {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	f.ctx, f.cancel = context.WithCancel(ctx)
	streamHandlers[f.address] = f.handler
	atomic.AddInt32(&listenersCount, 1)
	return nil
}

func (f *fakeStreamTransport) Stop(ctx context.Context) error {
	tcpMutex.Lock()
	defer tcpMutex.Unlock()

	f.cancel()
	streamHandlers[f.address] = nil
	atomic.AddInt32(&listenersCount, -1)
	return nil
}

func (f *fakeStreamTransport) Dial(ctx context.Context, address string) (io.ReadWriteCloser, error) {
	global.Debugf("fakeStreamTransport Dial from %s to %s", f.address, address)

	tcpMutex.RLock()
	defer tcpMutex.RUnlock()

	h := streamHandlers[address]

	if h == nil {
		return nil, errors.New("fakeStreamTransport: dial failed")
	}

	conn1, conn2 := net.Pipe()
	go h.HandleStream(f.ctx, f.address, conn2)

	return conn1, nil
}

func (f *fakeStreamTransport) Address() string {
	return f.address
}

// WaitFakeListeners wait for streamTransports will be started
// this is used to prevent race in tests
func WaitFakeListeners(count int32, timeout time.Duration) error {
	c := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func(ctx context.Context) {
		for ;atomic.LoadInt32(&listenersCount) != count && ctx.Err() == nil; {
			time.Sleep(time.Microsecond*100)
		}
		c <- ctx.Err()
	}(ctx)

	select {
		case err := <-c:
			return err
		case <-ctx.Done():
			return errors.W(ctx.Err(), "WaitFakeListeners timeout")
	}
}
