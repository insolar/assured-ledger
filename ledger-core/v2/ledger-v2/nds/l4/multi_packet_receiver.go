// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"bytes"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

type PacketReceiveFunc func(h apinetwork.Header, r *iokit.LimitedReader)

func ReceiveMultiplePackets(r io.ReadCloser, packetFn PacketReceiveFunc) {
	defer func() {
		_ = r.Close()
	}()

	for {
		ReceivePacket(r, )
	}
}

const MaxBufferedPacket = 4096

func ReceivePacket(r io.Reader, packetFn PacketReceiveFunc) (recoverable bool, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, MaxBufferedPacket))
	var h apinetwork.Header
	if err = h.DeserializeFrom(iokit.NewTeeReader(r, buf)); err != nil {
		return false, err
	}
	sz := uint64(0)
	if sz, err = h.GetPayloadLength(); err != nil {
		return false, err
	}

	if sz <= MaxBufferedPacket {
		if _, err = io.CopyN(buf, r, int64(sz)); err != nil {
			return false, err
		}
		packetFn(h, iokit.LimitReader(buf, int64(buf.Len())))
		return true, nil
	}

	iokit.PrependReader(buf.Bytes(), r))
}