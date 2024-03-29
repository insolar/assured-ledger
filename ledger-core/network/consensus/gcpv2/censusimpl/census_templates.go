package censusimpl

import (
	"context"
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/proofs"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

var _ localActiveCensus = &PrimingCensusTemplate{}

type copyToOnlinePopulation interface {
	copyToPopulation
	census.OnlinePopulation
}

func NewPrimingCensusForJoiner(localProfile profiles.StaticProfile, registries census.VersionedRegistries,
	vf cryptkit.SignatureVerifierFactory, allowEphemeral bool) *PrimingCensusTemplate {

	pop := NewJoinerPopulation(localProfile, vf)
	return newPrimingCensus(&pop, registries, allowEphemeral)
}

func NewPrimingCensus(intros []profiles.StaticProfile, localProfile profiles.StaticProfile, registries census.VersionedRegistries,
	vf cryptkit.SignatureVerifierFactory, allowEphemeral bool) *PrimingCensusTemplate {

	if len(intros) == 0 {
		panic("illegal state")
	}
	localID := localProfile.GetStaticNodeID()
	pop := NewManyNodePopulation(intros, localID, vf)
	return newPrimingCensus(&pop, registries, allowEphemeral)
}

func newPrimingCensus(pop copyToOnlinePopulation, registries census.VersionedRegistries, allowEphemeral bool) *PrimingCensusTemplate {

	var pd pulse.Data
	if allowEphemeral {
		pd.PulseEpoch = pulse.EphemeralPulseEpoch
	} else {
		pd = registries.GetVersionPulseData()
	}

	return &PrimingCensusTemplate{CensusTemplate{nil,
		pop, &evictedPopulation{}, pd, registries,
	}}
}

var _ census.Prime = &PrimingCensusTemplate{}

type PrimingCensusTemplate struct {
	CensusTemplate
}

func (c *PrimingCensusTemplate) onMadeActive() {
}

func (c *PrimingCensusTemplate) IsActive() bool {
	return c.chronicles.GetActiveCensus() == c
}

func (c *PrimingCensusTemplate) SetAsActiveTo(chronicles LocalConsensusChronicles) {
	if c.chronicles != nil {
		panic("illegal state")
	}
	lc := chronicles.(*localChronicles)
	c.chronicles = lc
	lc.makeActive(nil, c)
}

func (*PrimingCensusTemplate) GetCensusState() census.State {
	return census.PrimingCensus
}

func (c *PrimingCensusTemplate) GetExpectedPulseNumber() pulse.Number {
	switch {
	case c.pd.IsEmpty():
		return pulse.Unknown
	case c.pd.IsExpectedPulse():
		return c.pd.GetPulseNumber()
	}
	return c.pd.NextPulseNumber()
}

func (c *PrimingCensusTemplate) CreateBuilder(ctx context.Context, pn pulse.Number) census.Builder {

	if !pn.IsUnknown() && (!pn.IsTimePulse() || !c.GetExpectedPulseNumber().IsUnknownOrEqualTo(pn)) {
		panic("illegal value")
	}

	return newLocalCensusBuilder(ctx, c.chronicles, pn, c.online)
}

func (c *PrimingCensusTemplate) BuildCopy(pd pulse.Data, csh proofs.CloudStateHash, gsh proofs.GlobulaStateHash) census.Built {
	if csh == nil {
		panic("illegal value: CSH is nil")
	}
	if gsh == nil {
		panic("illegal value: GSH is nil")
	}

	if !c.pd.PulseEpoch.IsEphemeral() && pd.IsFromEphemeral() {
		panic("illegal value")
	}
	if !c.GetExpectedPulseNumber().IsUnknownOrEqualTo(pd.PulseNumber) {
		panic("illegal value")
	}

	return &BuiltCensusTemplate{ExpectedCensusTemplate{
		c.chronicles, c.online, c.evicted, c.chronicles.active,
		gsh, csh, pd.PulseNumber,
	}}
}

func (c *PrimingCensusTemplate) GetPulseNumber() pulse.Number {
	return c.pd.GetPulseNumber()
}

func (c *PrimingCensusTemplate) GetPulseData() pulse.Data {
	return c.pd
}

func (*PrimingCensusTemplate) GetGlobulaStateHash() proofs.GlobulaStateHash {
	return nil
}

func (*PrimingCensusTemplate) GetCloudStateHash() proofs.CloudStateHash {
	return nil
}

func (c PrimingCensusTemplate) String() string {
	return fmt.Sprintf("priming %s", c.CensusTemplate.String())
}

type CensusTemplate struct {
	chronicles *localChronicles
	online     copyToOnlinePopulation
	evicted    census.EvictedPopulation
	pd         pulse.Data

	registries census.VersionedRegistries
}

func (c *CensusTemplate) GetNearestPulseData() (bool, pulse.Data) {
	return true, c.pd
}

func (c *CensusTemplate) GetProfileFactory(ksf cryptkit.KeyStoreFactory) profiles.Factory {
	return c.chronicles.profileFactory
}

func (c *CensusTemplate) setVersionedRegistries(vr census.VersionedRegistries) {
	if vr == nil {
		panic("versioned registries are nil")
	}
	c.registries = vr
}

func (c *CensusTemplate) getVersionedRegistries() census.VersionedRegistries {
	return c.registries
}

func (c *CensusTemplate) GetOnlinePopulation() census.OnlinePopulation {
	return c.online
}

func (c *CensusTemplate) GetEvictedPopulation() census.EvictedPopulation {
	return c.evicted
}

func (c *CensusTemplate) GetOfflinePopulation() census.OfflinePopulation {
	return c.registries.GetOfflinePopulation()
}

func (c *CensusTemplate) GetMisbehaviorRegistry() census.MisbehaviorRegistry {
	return c.registries.GetMisbehaviorRegistry()
}

func (c *CensusTemplate) GetMandateRegistry() census.MandateRegistry {
	return c.registries.GetMandateRegistry()
}

func (c CensusTemplate) String() string {
	return fmt.Sprintf("pd:%v evicted:%v online:[%v]", c.pd, c.evicted, c.online)
}

var _ census.Active = &ActiveCensusTemplate{}

type ActiveCensusTemplate struct {
	CensusTemplate
	activeRef *ActiveCensusTemplate // hack for stringer
	gsh       proofs.GlobulaStateHash
	csh       proofs.CloudStateHash
}

func (c *ActiveCensusTemplate) onMadeActive() {
	c.activeRef = c
}

func (c *ActiveCensusTemplate) IsActive() bool {
	return c.chronicles.GetActiveCensus() == c
}

func (c *ActiveCensusTemplate) GetExpectedPulseNumber() pulse.Number {
	return c.pd.NextPulseNumber()
}

func (c *ActiveCensusTemplate) CreateBuilder(ctx context.Context, pn pulse.Number) census.Builder {

	if !pn.IsUnknown() && (!pn.IsTimePulse() || !c.GetExpectedPulseNumber().IsUnknownOrEqualTo(pn)) {
		panic("illegal value")
	}

	return newLocalCensusBuilder(ctx, c.chronicles, pn, c.online)
}

func (*ActiveCensusTemplate) GetCensusState() census.State {
	return census.SealedCensus
}

func (c *ActiveCensusTemplate) GetPulseNumber() pulse.Number {
	return c.pd.PulseNumber
}

func (c *ActiveCensusTemplate) GetPulseData() pulse.Data {
	return c.pd
}

func (c *ActiveCensusTemplate) GetGlobulaStateHash() proofs.GlobulaStateHash {
	return c.gsh
}

func (c *ActiveCensusTemplate) GetCloudStateHash() proofs.CloudStateHash {
	return c.csh
}

func (c ActiveCensusTemplate) String() string {
	mode := "active"
	if c.activeRef != c.chronicles.GetActiveCensus() {
		mode = "ex-active"
	}
	return fmt.Sprintf("%s %s gsh:%v csh:%v", mode, c.CensusTemplate.String(), c.gsh, c.csh)
}

var _ census.Expected = &ExpectedCensusTemplate{}

type ExpectedCensusTemplate struct {
	chronicles *localChronicles
	online     copyToOnlinePopulation
	evicted    census.EvictedPopulation
	prev       census.Active
	gsh        proofs.GlobulaStateHash
	csh        proofs.CloudStateHash
	pn         pulse.Number
}

func (c *ExpectedCensusTemplate) Rebuild(pn pulse.Number) census.Built {

	if pn.IsUnknown() || pn.IsTimePulse() && pn >= c.GetExpectedPulseNumber() {
		cp := BuiltCensusTemplate{*c}
		cp.expected.pn = pn
		return &cp
	}

	panic(fmt.Sprintf("illegal value: %v, %v", c.GetExpectedPulseNumber(), pn))
}

func (c *ExpectedCensusTemplate) GetNearestPulseData() (bool, pulse.Data) {
	return false, c.prev.GetPulseData()
}

func (c *ExpectedCensusTemplate) GetProfileFactory(ksf cryptkit.KeyStoreFactory) profiles.Factory {
	return c.chronicles.GetProfileFactory(ksf)
}

func (c *ExpectedCensusTemplate) GetEvictedPopulation() census.EvictedPopulation {
	return c.evicted
}

func (c *ExpectedCensusTemplate) GetExpectedPulseNumber() pulse.Number {
	return c.pn
}

func (c *ExpectedCensusTemplate) GetCensusState() census.State {
	return census.CompleteCensus
}

func (c *ExpectedCensusTemplate) GetPulseNumber() pulse.Number {
	return c.pn
}

func (c *ExpectedCensusTemplate) GetGlobulaStateHash() proofs.GlobulaStateHash {
	return c.gsh
}

func (c *ExpectedCensusTemplate) GetCloudStateHash() proofs.CloudStateHash {
	return c.csh
}

func (c *ExpectedCensusTemplate) GetOnlinePopulation() census.OnlinePopulation {
	return c.online
}

func (c *ExpectedCensusTemplate) GetOfflinePopulation() census.OfflinePopulation {
	// TODO Should be provided via relevant builder
	return c.prev.GetOfflinePopulation()
}

func (c *ExpectedCensusTemplate) GetMisbehaviorRegistry() census.MisbehaviorRegistry {
	// TODO Should be provided via relevant builder
	return c.prev.GetMisbehaviorRegistry()
}

func (c *ExpectedCensusTemplate) GetMandateRegistry() census.MandateRegistry {
	// TODO Should be provided via relevant builder
	return c.prev.GetMandateRegistry()
}

func (c *ExpectedCensusTemplate) CreateBuilder(ctx context.Context, pn pulse.Number) census.Builder {
	return newLocalCensusBuilder(ctx, c.chronicles, pn, c.online)
}

func (c *ExpectedCensusTemplate) GetPrevious() census.Active {
	return c.prev
}

func (c *ExpectedCensusTemplate) MakeActive(pd pulse.Data) census.Active {

	a := ActiveCensusTemplate{
		CensusTemplate{
			c.chronicles, c.online, c.evicted, pd, nil,
		}, nil,
		c.gsh,
		c.csh,
	}

	c.chronicles.makeActive(c, &a)
	return &a
}

func (c *ExpectedCensusTemplate) IsActive() bool {
	return false
}

func (c ExpectedCensusTemplate) String() string {
	return fmt.Sprintf("expected pn:%v evicted:%v online:[%v] gsh:%v csh:%v", c.pn, c.evicted, c.online, c.gsh, c.csh)
}

var _ census.Built = &BuiltCensusTemplate{}

type BuiltCensusTemplate struct {
	expected ExpectedCensusTemplate
}

func (b *BuiltCensusTemplate) Update(csh proofs.CloudStateHash, gsh proofs.GlobulaStateHash) census.Built {
	if b.expected.gsh != nil && gsh == nil {
		panic("illegal value")
	}
	if b.expected.csh != nil && csh == nil {
		panic("illegal value")
	}
	cp := *b
	cp.expected.gsh = gsh
	cp.expected.csh = csh
	return &cp
}

func (b *BuiltCensusTemplate) GetOnlinePopulation() census.OnlinePopulation {
	return b.expected.GetOnlinePopulation()
}

func (b *BuiltCensusTemplate) GetEvictedPopulation() census.EvictedPopulation {
	return b.expected.GetEvictedPopulation()
}

func (b *BuiltCensusTemplate) GetGlobulaStateHash() proofs.GlobulaStateHash {
	return b.expected.GetGlobulaStateHash()
}

func (b *BuiltCensusTemplate) GetCloudStateHash() proofs.CloudStateHash {
	return b.expected.GetCloudStateHash()
}

func (b *BuiltCensusTemplate) GetNearestPulseData() (bool, pulse.Data) {
	return b.expected.GetNearestPulseData()
}

func (b *BuiltCensusTemplate) MakeExpected() census.Expected {
	return b.expected.chronicles.makeExpected(&b.expected)
}

func (b BuiltCensusTemplate) String() string {
	return fmt.Sprintf("built-%s", b.expected.String())
}
