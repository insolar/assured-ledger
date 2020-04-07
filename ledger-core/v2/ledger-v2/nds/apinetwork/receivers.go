// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
)

type PacketReceiveFunc func(h Header, r iokit.LenReader)

func ReceiveMultiplePackets(r io.ReadCloser, packetFn PacketReceiveFunc, maxBufferedPacket uint16) {
	defer func() {
		_ = r.Close()
	}()

	for {
		if err := ReceivePacket(r, packetFn, maxBufferedPacket); err != nil {
			packetFn(Header{}, iokit.WrapError(err))
			return
		}
	}
}

func ReceivePacket(r io.Reader, packetFn PacketReceiveFunc, maxBufferedPacket uint16) error {
	buf := bytes.NewBuffer(make([]byte, 0, maxBufferedPacket))
	var h Header

	err := h.DeserializeFrom(iokit.NewTeeReader(r, buf))
	if err != nil {
		return err
	}
	sz := uint64(0)
	if sz, err = h.GetPayloadLength(); err != nil {
		return err
	}

	if sz+uint64(h.ByteSize()) <= uint64(maxBufferedPacket) {
		if _, err = io.CopyN(buf, r, int64(sz)); err != nil {
			return err
		}
		packetFn(h, buf)
		return nil
	}

	mx := sync.Mutex{}
	mx.Lock()
	sr := iokit.LimitReaderWithTrigger(iokit.PrependReader(buf.Bytes(), r), int64(buf.Len())+int64(sz), func(n int64) {
		if n > 0 {
			err = errors.New("incomplete read")
		}
		mx.Unlock()
	})
	packetFn(h, sr)
	mx.Lock()
	mx.Unlock()
	return err
}
