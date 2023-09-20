package dropbag

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/drafts/dropbag/dbcommon"
	"github.com/insolar/assured-ledger/ledger-core/drafts/dropbag/dbsv1"
)

func OpenReadStorage(sr dbcommon.StorageSeqReader, config dbcommon.ReadConfig, payloadFactory dbcommon.PayloadFactory) (dbcommon.PayloadBuilder, error) {
	f, opt, err := dbcommon.ReadFormatAndOptions(sr)
	if err != nil {
		return nil, err
	}

	var pb dbcommon.PayloadBuilder
	switch f {
	case 1:
		pb, err = dbsv1.OpenReadStorage(sr, payloadFactory, config, opt)
	default:
		return nil, fmt.Errorf("unknown storage format: format%x", f)
	}

	if pb == nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read storage: format=%x", f)
	}

	if err == nil {
		err = pb.Finished()
	}
	if err != nil {
		err = pb.Failed(err)
	}
	return pb, err
}

func OpenWriteStorage(sw dbcommon.StorageSeqWriter, f dbcommon.FileFormat, options dbcommon.FormatOptions) (dbcommon.PayloadWriter, error) {
	var pw dbcommon.PayloadWriter
	var err error

	switch f {
	case 1:
		pw, err = dbsv1.PrepareWriteStorage(sw, options)
	default:
		return nil, fmt.Errorf("unknown storage format: format%x", f)
	}
	if pw == nil {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to write storage: format=%x", f)
	}

	if err != nil {
		_ = pw.Close()
		return nil, err
	}

	if err := dbcommon.WriteFormatAndOptions(sw, f, options); err != nil {
		_ = pw.Close()
		return nil, err
	}
	return pw, nil
}
