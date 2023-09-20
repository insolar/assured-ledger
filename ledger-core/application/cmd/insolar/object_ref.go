package main

import "github.com/insolar/assured-ledger/ledger-core/reference"

// isObjectReferenceString checks the validity of the reference
// deprecated
func isObjectReferenceString(input string) bool {
	_, err := reference.GlobalObjectFromString(input)
	return err == nil
}
