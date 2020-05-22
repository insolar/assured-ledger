// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/entropy"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
)

// In reality compares no sort vs with sort + in/out conversions of array of empty interfaces
// This benchamark results would be suitable for analyzing how much we lost on input/output
// conversion only after sorting removal
//
// prepare benchmarks results:
// go test -v ./ledger/jetcoordinator/ -bench=SelectByEntropy -cpu=1 -benchmem -run=NONE > wrapped.txt
// SelectByEntropyBench=orig go test -v ./ledger/jetcoordinator/ -bench=SelectByEntropy -cpu=1 -benchmem -run=NONE > orig.txt
//
// measure overhead:
// benchcmp orig.txt wrapped.txt
//
func BenchmarkSelectByEntropy(b *testing.B) {
	benchtype := strings.ToLower(os.Getenv("SelectByEntropyBench"))
	switch benchtype {
	case "orig", "wrapped":
		// all ok
	case "":
		benchtype = "wrapped"
	default:
		panic(fmt.Sprintf("Unknown benchtype %v", benchtype))
	}

	benches := []struct {
		values int
		count  int
	}{
		{10, 1},
		{10, 5},
		{10, 10},
		{100, 1},
		{100, 50},
		{100, 100},
		{1000, 1},
		{1000, 500},
		{1000, 1000},
	}
	fmt.Printf("# Bench: %v\n", benchtype)
	for _, bench := range benches {
		b.Run(
			fmt.Sprintf("%v_from_%v", bench.count, bench.values),
			func(b *testing.B) {
				if benchtype == "orig" {
					benchSelectByEntropy(b, bench.values, bench.count)
					return
				}
				benchSelectByEntropyWrapped(b, bench.values, bench.count)
			})
	}
}

// compiler should avoid to optimize call of benched function
var resultsI []interface{}
var resultsB [][]byte

func benchSelectByEntropy(b *testing.B, valuescount int, count int) {
	scheme := platformpolicy.NewPlatformCryptographyScheme()
	entropybytes := randslice(64)

	values := make([]interface{}, 0, valuescount)
	for i := 0; i < valuescount; i++ {
		values = append(values, interface{}(randslice(64)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// sort.SliceStable(valuesB, )
		// bytes.Compare(a, b) < 0
		resultsI, _ = entropy.SelectByEntropy(scheme, entropybytes, values, count)
	}
}

// compiler should avoid to optimize call of benched function
var refresults []reference.Global

func benchSelectByEntropyWrapped(b *testing.B, valuescount int, count int) {
	scheme := platformpolicy.NewPlatformCryptographyScheme()

	var e pulsestor.Entropy
	copy(e[:], randslice(64))

	values := make([]node.Node, 0, valuescount)
	for i := 0; i < valuescount; i++ {
		values = append(values, node.Node{ID: gen.UniqueReference()})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		refresults, _ = getRefs(scheme, e[:], values, count)
	}
}

func randslice(size int) []byte {
	b := make([]byte, size)
	rand.Read(b)
	return b
}
