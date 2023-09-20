package censusimpl

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func BenchmarkMetricAssignment(b *testing.B) {
	for _, count := range []int{1, 10, 100, 1000} {
		nodeCount := count
		b.Run(fmt.Sprint(nodeCount), func(b *testing.B) {
			pop := createManyNodePopulation(b, nodeCount)
			role := pop.GetRolePopulation(member.PrimaryRoleVirtual)

			stats := make([]uint64, nodeCount + 1)

			b.Run("byCount", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := b.N; i > 0; i-- {
					assigned, _ := role.GetAssignmentByCount(1597334677*uint64(i), 0)
					stats[assigned.GetNodeID()]++
				}
			})

			checkDistribution(b, stats, false)
			stats = make([]uint64, nodeCount + 1)

			b.Run("byPower", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := b.N; i > 0; i-- {
					assigned, _ := role.GetAssignmentByPower(1597334677*uint64(i), 0)
					stats[assigned.GetNodeID()]++
				}
			})

			checkDistribution(b, stats, true)
		})
	}
}

func checkDistribution(t *testing.B, stats []uint64, power bool) {
	stats = stats[1:] // zero index is unused

	total := float64(stats[0])
	min, max := total, total
	for _, vs := range stats[1:] {
		v := float64(vs)
		// if power {
		// 	v /= float64(i + 1)
		// }
		total += v

		switch {
		case v < min:
			min = v
		case v > max:
			max = v
		}
	}
	expected := total / float64(len(stats))

	// fmt.Println()
	// for _, vs := range stats[1:] {
	// 	fmt.Printf("%.2f ", float64(vs)/expected)
	// }
	// fmt.Println()

	name := "byCount"
	if power {
		name = "byPower"
	}

	t.Logf("%s: minDev: %.2f maxDev: %.2f", name, min/expected, max/expected)
}

func BenchmarkVirtualAssignment(b *testing.B) {
	benchmarkVirtualAssignment(b, "xor", calcVirtualMetricByEntropyXor)
	benchmarkVirtualAssignment(b, "sha", calcVirtualMetricByEntropySha)
}

func benchmarkVirtualAssignment(b *testing.B, name string, metricFn func(entropy longbits.Bits256, base reference.Local) uint64) {
	var entropy longbits.Bits256
	rand.Read(entropy[:])

	refs := make([]reference.Local, 1<<16)
	for i := range refs {
		refs[i] = gen.UniqueLocalRef()
	}

	for _, count := range []int{1, 10, 100, 1000} {
		nodeCount := count
		b.Run(fmt.Sprintf("%d/%s", nodeCount, name), func(b *testing.B) {
			pop := createManyNodePopulation(b, nodeCount)
			role := pop.GetRolePopulation(member.PrimaryRoleVirtual)

			stats := make([]uint64, nodeCount + 1)

			b.Run("byCount", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := b.N; i > 0; i-- {
					ref := refs[i%len(refs)]
					metric := metricFn(entropy, ref)
					assigned, _ := role.GetAssignmentByCount(metric, 0)
					stats[assigned.GetNodeID()]++
				}
			})

			checkDistribution(b, stats, false)
			stats = make([]uint64, nodeCount + 1)

			b.Run("byPower", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := b.N; i > 0; i-- {
					ref := refs[i%len(refs)]
					metric := metricFn(entropy, ref)
					assigned, _ := role.GetAssignmentByPower(metric, 0)
					stats[assigned.GetNodeID()]++
				}
			})

			checkDistribution(b, stats, true)
		})
	}
}

func calcVirtualMetricByEntropySha(entropy longbits.Bits256, base reference.Local) uint64 {
	h := sha256.New()
	h.Write(base.AsBytes())
	h.Write(entropy[:])
	metric := longbits.CutOutUint64(h.Sum(nil))
	return metric
}

func calcVirtualMetricByEntropyXor(entropy longbits.Bits256, base reference.Local) uint64 {
	b := base.AsBytes()
	xorBytes(b, entropy[:])
	metric := longbits.CutOutUint64(b)
	return metric
}

func xorBytes(dest []byte, b []byte) {
	n := len(dest)
	for i := range b {
		dest[i % n] ^= b[i]
	}
}

func createManyNodePopulation(t minimock.Tester, count int) census.OnlinePopulation {
	pr := make([]profiles.StaticProfile, 0, count)

	id := node.ShortNodeID(0)
	for ;count > 0; count-- {
		id++
		sp := createStaticProfile(t, id, reference.NewSelf(gen.UniqueLocalRef()))
		pr = append(pr, sp)
	}

	svf := cryptkit.NewSignatureVerifierFactoryMock(t)
	svf.CreateSignatureVerifierWithPKSMock.Return(nil)

	op := NewManyNodePopulation(pr, id, svf)

	// Following processing mimics consensus behavior on node ordering.
	// Ordering rules must be followed strictly.
	// WARNING! This code will only work properly when all nodes have same PrimaryRole and non-zero power

	dp := NewDynamicPopulationCopySelf(&op)
	for _, np := range op.GetProfiles() {
		if np.GetNodeID() != id  {
			dp.AddProfile(np.GetStatic())
		}
	}

	pfs := dp.GetUnorderedProfiles()
	for _, np := range pfs {
		np.SetOpMode(member.ModeNormal)
		pw := np.GetStatic().GetStartPower()
		np.SetPower(pw)
	}

	sort.Slice(pfs, func(i, j int) bool {
		// Power sorting is REVERSED
		return pfs[j].GetDeclaredPower() < pfs[i].GetDeclaredPower()
	})

	idx := member.AsIndex(0)
	for _, np := range pfs {
		np.SetIndex(idx)
		idx++
	}

	ap, _ := dp.CopyAndSeparate(false, nil)
	return ap
}

func createStaticProfile(t minimock.Tester, localNode node.ShortNodeID, localRef reference.Global) profiles.StaticProfile {
	cp := profiles.NewCandidateProfileMock(t)
	cp.GetBriefIntroSignedDigestMock.Return(cryptkit.SignedDigest{})
	cp.GetDefaultEndpointMock.Return(nil)
	cp.GetExtraEndpointsMock.Return(nil)
	cp.GetIssuedAtPulseMock.Return(pulse.MinTimePulse)
	cp.GetIssuedAtTimeMock.Return(time.Now())
	cp.GetIssuerIDMock.Return(localNode)
	cp.GetIssuerSignatureMock.Return(cryptkit.Signature{})
	cp.GetNodePublicKeyMock.Return(cryptkit.NewSigningKeyHolderMock(t))
	cp.GetPowerLevelsMock.Return(member.PowerSet{0, 0, 0, 255})
	cp.GetPrimaryRoleMock.Return(member.PrimaryRoleVirtual)
	cp.GetReferenceMock.Return(localRef)
	cp.GetSpecialRolesMock.Return(0)
	cp.GetStartPowerMock.Return(member.PowerOf(uint16(localNode)))
	cp.GetStaticNodeIDMock.Return(localNode)

	return profiles.NewStaticProfileByFull(cp, nil)
}
