#!/usr/bin/env sh
# filters coverage stats for code that could be tested, but should not affect coverage metric
#
# * generated code (mock and stringer)
# * command line tools code
# * test utils
grep -v "_mock.go:" | \
    grep -v "_string.go:" | \
    grep -v "_gen.go:" | \
    grep -v 'github.com/insolar/assured-ledger/ledger-core/v2/cmd/' | \
    grep -v "github.com/insolar/assured-ledger/ledger-core/v2/testutils" | \
    grep -v "storage/storagetest" | \
    grep -v ".pb.go:"
