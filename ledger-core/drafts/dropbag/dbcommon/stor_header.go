package dbcommon

import (
	"io"
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
)

type PayloadFactory interface {
	CreatePayloadBuilder(format FileFormat, sr StorageSeqReader) PayloadBuilder
}

type ChapterDetails struct {
	// [1..)
	Index   int
	Options uint32
	// [0..2024]
	Type uint16
}

type StorageEntryPosition struct {
	StorageOffset int64
	ByteOffset    int
}

type PayloadBuilder interface {
	AddPrelude(bytes []byte, pos StorageEntryPosition) error
	AddConclude(bytes []byte, pos StorageEntryPosition, totalCount uint32) error
	AddChapter(bytes []byte, pos StorageEntryPosition, details ChapterDetails) error
	NeedsNextChapter() bool

	Finished() error
	Failed(error) error
}

type PayloadWriter interface {
	io.Closer
	WritePrelude(bytes []byte, concludeMaxLength int) (StorageEntryPosition, error)
	WriteConclude(bytes []byte) (StorageEntryPosition, error)
	WriteChapter(bytes []byte, details ChapterDetails) (StorageEntryPosition, error)
}

type FileFormat uint16
type FormatOptions uint64

const formatFieldId = 16

var formatField = protokit.WireFixed64.Tag(formatFieldId)

type ReadConfig struct {
	ReadAllEntries bool
	AlwaysCopy     bool
	//StorageOptions FormatOptions
}

func ReadFormatAndOptions(sr StorageSeqReader) (FileFormat, FormatOptions, error) {
	if v, err := formatField.ReadTagValue(sr); err != nil {
		return 0, 0, err
	} else {
		return FileFormat(v & math.MaxUint16), FormatOptions(v >> 16), nil
	}
}

func WriteFormatAndOptions(sw StorageSeqWriter, format FileFormat, options FormatOptions) error {
	return formatField.WriteTagValue(sw, uint64(format)|uint64(options)<<16)
}

type StorageSeqReader interface {
	io.ByteReader
	io.Reader
	io.Seeker
	CanSeek() bool
	CanReadMapped() bool
	Offset() int64
	ReadMapped(n int64) ([]byte, error)
}

type StorageBlockReader interface {
	io.ReaderAt
	io.Closer
	ReadAtMapped(n int64, off int64) ([]byte, error)
}

type StorageSeqWriter interface {
	io.ByteWriter
	io.Writer
	Offset() int64
}
