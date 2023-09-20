package l1

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

func NewTLS(binding nwapi.Address, preference nwapi.Preference, config *tls.Config) SessionfulTransportProvider {
	return tlsProvider{addr: binding.AsTCPAddr(), preference: preference, config: config}
}

type tlsProvider struct {
	addr       net.TCPAddr
	config     *tls.Config
	preference nwapi.Preference
}

func (v tlsProvider) IsZero() bool {
	return v.addr.IP == nil
}

func (v tlsProvider) CreateListeningFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, nwapi.Address, error) {
	switch {
	case receiveFn == nil:
		panic(throw.IllegalValue())
	case v.IsZero():
		panic(throw.IllegalState())
	case len(v.config.Certificates) > 0 || v.config.GetCertificate != nil || v.config.GetConfigForClient != nil:
		// ok
	default:
		// mimics tls.CreateListeningFactory
		return nil, nwapi.Address{}, errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
	}

	tcpConn, err := ListenTCPWithReuse("tcp", &v.addr)
	if err != nil {
		return nil, nwapi.Address{}, err
	}

	localAddr := *tcpConn.Addr().(*net.TCPAddr)

	tlsConn := tls.NewListener(tcpConn, v.config)
	t := &tlsTransportFactory{tlsConn, receiveFn, v.config, localAddr, v.preference}

	go runTCPListener(tlsConn, t.tlsConnect)
	return t, nwapi.FromTCPAddr(&localAddr), nil
}

func (v tlsProvider) CreateOutgoingOnlyFactory(receiveFn SessionfulConnectFunc) (OutTransportFactory, error) {
	return &tlsTransportFactory{nil, receiveFn, v.config, v.addr, v.preference}, nil
}

func (v tlsProvider) Close() error {
	return nil
}

/*********************************/

type tlsTransportFactory struct {
	listener   net.Listener
	receiveFn  SessionfulConnectFunc
	config     *tls.Config
	addr       net.TCPAddr
	preference nwapi.Preference
}

func (p *tlsTransportFactory) IsZero() bool {
	return p.addr.IP == nil
}

func (p *tlsTransportFactory) Close() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

func (p *tlsTransportFactory) ConnectTo(to nwapi.Address) (OneWayTransport, error) {
	return p.ConnectToExt(to, nil)
}

func (p *tlsTransportFactory) ConnectToExt(to nwapi.Address, peerVerify VerifyPeerCertificateFunc) (OneWayTransport, error) {
	if !to.IsNetCompatible() {
		return nil, nil
	}

	var err error
	to, err = to.Resolve(context.Background(), net.DefaultResolver, p.preference)
	if err != nil {
		return nil, err
	}

	peerConfig := p.config
	if peerVerify != nil {
		cfg := p.config.Clone()
		cfg.VerifyPeerCertificate = peerVerify
		peerConfig = cfg
	}


	var conn *tls.Conn

	if p.addr.Port != 0 {
		if conn, err = tls.DialWithDialer(DialerTCPWithReuse(&p.addr), "tcp", to.String(), peerConfig); err != nil {
			if !isReusePortError(err) {
				return nil, err
			}
			lAddr := p.addr
			lAddr.Port = 0
			conn, err = tls.DialWithDialer(&net.Dialer{LocalAddr: &lAddr}, "tcp", to.String(), peerConfig)
		}
	} else {
		conn, err = tls.DialWithDialer(&net.Dialer{LocalAddr: &p.addr}, "tcp", to.String(), peerConfig)
	}

	if err != nil {
		return nil, err
	}

	// force connection setup now
	if err = conn.Handshake(); err != nil {
		return nil, err
	} else if err = p.checkProtos(conn); err != nil {
		return nil, err
	}

	tcpOut := tcpOutTransport{conn, nil, 0}

	if p.receiveFn == nil {
		return &tcpOut, nil
	}
	return &tcpSemiTransport{tcpOut, p.receiveFn}, nil
}

func (p *tlsTransportFactory) tlsConnect(local, remote nwapi.Address, conn io.ReadWriteCloser, w OneWayTransport, err error) bool {
	if err != nil {
		return p.receiveFn(local, remote, conn, w, err)
	}

	tlsConn := conn.(*tls.Conn)

	if err = tlsConn.Handshake(); err != nil {
		// If the handshake failed due to the client not speaking
		// TLS, assume they're speaking plaintext HTTP and write a
		// 400 response on the TLS conn's underlying net.TwoWayConn.
		if re, ok := err.(tls.RecordHeaderError); ok && re.Conn != nil && tlsRecordHeaderLooksLikeHTTP(re.RecordHeader) {
			_, _ = io.WriteString(re.Conn, "HTTP/1.0 400 Bad Request\r\n\r\nClient sent an HTTP request to an HTTPS server.\n")
			_ = re.Conn.Close()
			return true
		}
	} else if err = p.checkProtos(tlsConn); err == nil {
		return p.receiveFn(local, remote, conn, w, nil)
	}

	_ = conn.Close()

	// err = throw.WithDetails(err, l2.ConnErrDetails{Local:local, Remote:remote})
	return p.receiveFn(local, remote, nil, nil, err)
}

func (p *tlsTransportFactory) checkProtos(tlsConn *tls.Conn) error {
	if len(p.config.NextProtos) == 0 {
		return nil
	}
	tlsState := tlsConn.ConnectionState()
	for _, proto := range p.config.NextProtos {
		if proto == tlsState.NegotiatedProtocol {
			return nil
		}
	}
	return errors.New("unmatched TLS application level protocol")
}

// tlsRecordHeaderLooksLikeHTTP reports whether a TLS record header
// looks like it might've been a misdirected plaintext HTTP request.
func tlsRecordHeaderLooksLikeHTTP(hdr [5]byte) bool {
	switch string(hdr[:]) {
	case "GET /", "HEAD ", "POST ", "PUT /", "OPTIO":
		return true
	}
	return false
}
