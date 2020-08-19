// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package census

import (
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/common/endpoints"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census.OfflinePopulation -o . -s _mock.go -g

type OfflinePopulation interface {
	FindRegisteredProfile(identity endpoints.Inbound) profiles.Host
	// FindPulsarProfile(pulsarId PulsarId) PulsarProfile
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census.OnlinePopulation -o . -s _mock.go -g

type OnlinePopulation interface {
	FindProfile(nodeID node.ShortNodeID) profiles.ActiveNode

	// IsValid indicates that this population was built without issues
	IsValid() bool
	// IsClean indicates that this population is both valid and does not contain non-operational nodes
	IsClean() bool

	// GetIndexedCount gets a number of indexed (valid) nodes in the population
	GetIndexedCount() int
	// GetIndexedCapacity indicates the expected size of population.
	// when IsValid()==true then GetIndexedCapacity() == GetIndexedCount(), otherwise GetIndexedCapacity() >= GetIndexedCount()
	GetIndexedCapacity() int

	// GetSuspendedCount returns a number of suspended nodes in the population
	GetSuspendedCount() int
	// GetMistrustedCount returns a number of mistrusted nodes in the population
	GetMistrustedCount() int

	// GetRolePopulation returns nil for (1) PrimaryRoleInactive, (2) for roles without any members either working or idle
	GetRolePopulation(role member.PrimaryRole) RolePopulation

	// GetIdleProfiles returns a list of idle members, irrelevant of their roles. Will return nil when !IsValid().
	// Returned slice doesn't contain nil.
	GetIdleProfiles() []profiles.ActiveNode

	// GetPoweredProfiles returns a list of non-idle members, irrelevant of their roles. Will return nil when !IsValid().
	// Returned slice doesn't contain nil.
	GetPoweredProfiles() []profiles.ActiveNode

	// GetIdleCount returns a total number of idle members in the population, irrelevant of their roles.
	GetIdleCount() int

	// GetPoweredRoles returns a sorted (starting from the highest role) list of roles with non-idle members.
	GetPoweredRoles() []member.PrimaryRole

	// GetProfiles returns a list of nodes of GetIndexedCapacity() size, will contain nil values when GetIndexedCapacity() > GetIndexedCount()
	GetProfiles() []profiles.ActiveNode

	// GetProfile returns nil or node for the given index.
	GetProfile(index member.Index) profiles.ActiveNode

	// GetLocalProfile returns local node, can be nil when !IsValid().
	GetLocalProfile() profiles.LocalNode
}

type RecoverableErrorTypes uint32

const EmptyPopulation RecoverableErrorTypes = 0

const (
	External RecoverableErrorTypes = 1 << iota
	EmptySlot
	IllegalRole
	IllegalMode
	IllegalIndex
	DuplicateIndex
	BriefProfile
	DuplicateID
	IllegalSorting
	MissingSelf
)

func (v RecoverableErrorTypes) String() string {
	b := strings.Builder{}
	b.WriteRune('[')
	appendByBit(&b, &v, "External")
	appendByBit(&b, &v, "EmptySlot")
	appendByBit(&b, &v, "IllegalRole")
	appendByBit(&b, &v, "IllegalMode")
	appendByBit(&b, &v, "IllegalIndex")
	appendByBit(&b, &v, "DuplicateIndex")
	appendByBit(&b, &v, "BriefProfile")
	appendByBit(&b, &v, "DuplicateID")
	appendByBit(&b, &v, "IllegalSorting")
	appendByBit(&b, &v, "MissingSelf")
	b.WriteRune(']')

	return b.String()
}

func appendByBit(b *strings.Builder, v *RecoverableErrorTypes, s string) {
	if *v&1 != 0 {
		b.WriteString(s)
		b.WriteByte(' ')
	}
	*v >>= 1
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census.EvictedPopulation -o . -s _mock.go -g

type EvictedPopulation interface {
	/* when the relevant online population is !IsValid() then not all nodes can be accessed by nodeID */
	FindProfile(nodeID node.ShortNodeID) profiles.EvictedNode
	GetCount() int
	/* slice will never contain nil. when the relevant online population is !IsValid() then it will also include erroneous nodes */
	GetProfiles() []profiles.EvictedNode

	IsValid() bool
	GetDetectedErrors() RecoverableErrorTypes
}

type PopulationBuilder interface {
	GetCount() int
	// SetCapacity
	AddProfile(intro profiles.StaticProfile) profiles.Updatable
	RemoveProfile(nodeID node.ShortNodeID)
	GetUnorderedProfiles() []profiles.Updatable
	FindProfile(nodeID node.ShortNodeID) profiles.Updatable
	GetLocalProfile() profiles.Updatable
	RemoveOthers()
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census.RolePopulation -o . -s _mock.go -g

type RolePopulation interface {
	// GetPrimaryRole returns a role of all nodes of this population, != PrimaryRoleInactive
	GetPrimaryRole() member.PrimaryRole
	// IsValid returns true when the relevant population is valid and there are some members in this role
	IsValid() bool
	// GetWorkingPower returns total power of all members in this role
	GetWorkingPower() uint32
	// GetWorkingCount returns total number of working (powered) members in this role
	GetWorkingCount() int
	// GetIdleCount returns total number of idle (non-powered) members in this role
	GetIdleCount() int

	// GetProfiles gives a list of working members in this role, will return nil when !IsValid() or GetWorkingPower()==0 */
	GetProfiles() []profiles.ActiveNode

	/*
		Returns a member (assigned) that can be assigned to to a task with the given (metric).
		It does flat distribution (all members with non-zero power are considered of same weight).

		If a default distribution falls the a member given as excludeID, then such member will be returned as (excluded)
		and the function will provide an alternative member as (assigned).

		When it was not possible to provide an alternative member then the same member will be returned as (assigned) and (excluded).

		When population is empty or invalid, then (nil, nil) is returned.
	*/
	GetAssignmentByCount(metric uint64, excludeID node.ShortNodeID) (assigned, excluded profiles.ActiveNode)
	/*
		Similar to GetAssignmentByCount, but it does weighed distribution across non-zero power members based on member's power.
	*/
	GetAssignmentByPower(metric uint64, excludeID node.ShortNodeID) (assigned, excluded profiles.ActiveNode)
}


type AssignmentFunc = func(metric uint64, excludeID node.ShortNodeID) (assigned, excluded profiles.ActiveNode)
