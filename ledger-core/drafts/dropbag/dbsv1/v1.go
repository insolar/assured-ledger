package dbsv1

import (
	"github.com/insolar/assured-ledger/ledger-core/drafts/dropbag/dbcommon"
)

const FormatId dbcommon.FileFormat = 1

func OpenReadStorage(sr dbcommon.StorageSeqReader, payloadFactory dbcommon.PayloadFactory,
	config dbcommon.ReadConfig, options dbcommon.FormatOptions,
) (dbcommon.PayloadBuilder, error) {
	pb := payloadFactory.CreatePayloadBuilder(FormatId, sr)
	v1 := StorageFileV1Reader{Config: config, Builder: pb}
	v1.StorageOptions = options
	return pb, v1.Read(sr)
}

func PrepareWriteStorage(sw dbcommon.StorageSeqWriter, options dbcommon.FormatOptions) (dbcommon.PayloadWriter, error) {
	return NewStorageFileV1Writer(sw, StorageFileV1{StorageOptions: options})
}
