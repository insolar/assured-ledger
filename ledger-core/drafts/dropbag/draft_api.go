package dropbag

import (
	"github.com/insolar/assured-ledger/ledger-core/drafts/dropbag/dbcommon"
)

type DropStorageBuilder interface {
}

type DropLifeline interface {
	// consists of 1xDropOpening, 1xDropClosing, 1+ DropRevisions
	// DropRevision keeps info on rearrangements

	// TODO appending summary info updates - needs something simple and cheap
}

// ==================

type EntryStorageCabinet interface {
	// Has ControlSection that keeps DropLifelines etc
	// Jet trees
	// Node lists
}

type EntryStorageShelf interface {
	// one per section type per EntryStorageCabinet
	// Consists of EntryCollections:
	// - directory (keys + brief info) partitioned by drops // can have one storage file per partition when is written, can be combined later
	// - alt_directory (keys + brief info but by using an alternative cryptography scheme)
	// - content (record data) // one per shelf, accessed by index+ofs+len

	// hides differences for read-only collections and collections being written
}

type EntryCollection interface {
	// index-based access

	// hides differences:
	// - lazy and non-lazy read implementation of an indexed set
	// - set being built
}

type EntryStorageAdapter interface {
	// provides support lazy / packed / open-read-close access to physical storage
}

// ================== file specific implementation

type StorageFileFolder interface {
}

type StorageFile interface {
}

type StorageFileReader interface {
}

type StorageFormatAdapter interface {
	// checks individual entry CRC
	// checks file on reopening
	// facilitates read of lazy entries -> need to know format
}

type StorageURI string
type StorageReadAdapter interface {
	GetURI() StorageURI

	OpenForSeqRead() dbcommon.StorageSeqReader
	OpenForBlockRead() dbcommon.StorageBlockReader
}
