package keyset

import "github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"

type Key = longbits.ByteString

// A basic set of keys, that can be wrapped by an extra keyset logic.
type KeyList interface {
	// returns true when the given key is within the set
	Contains(Key) bool
	// lists keys
	EnumKeys(func(k Key) bool) bool
	// number of unique keys
	Count() int
}

// An advanced set of keys, that also represent an open set (tracks exclusions, not inclusions)
//
// NB! An immutable inclusive/closed set MUST be able to cast to KeyList & InclusiveKeySet.
// An open or a mutable KeySet MUST NOT be able to cast to KeyList & InclusiveKeySet.
// This behavior is supported by this package.
//
type KeySet interface {
	// returns true when this set is empty
	IsNothing() bool
	// returns true when this set matches everything
	IsEverything() bool
	// returns true when the set is open / unbound and only Contains exclusions
	IsOpenSet() bool
	// returns true when the given key is within the set
	Contains(Key) bool
	// returns true when any key of the given set is within this set
	ContainsAny(KeySet) bool

	// returns true when this set Contains all keys from the given one
	SupersetOf(KeySet) bool
	// returns true when all keys of this set are contained in the given one
	SubsetOf(KeySet) bool
	// returns true when both sets have same set of keys
	Equal(KeySet) bool
	// a faster equivalent of Equal(ks.Inverse())
	EqualInverse(KeySet) bool

	// returns a set that has all keys but this set
	Inverse() KeySet
	// returns a set of keys present in at least one sets
	Union(KeySet) KeySet
	// returns a set of keys present in both sets
	Intersect(KeySet) KeySet
	// returns a set of keys present in this set and not present in the given set
	Subtract(KeySet) KeySet

	// WARNING! Do not use
	// lists keys (when IsOpenSet() == true, as then it lists _excluded_ keys)
	EnumRawKeys(func(k Key, exclusive bool) bool) bool
	// WARNING! Do not use. This must NOT be treated as a size of a set.
	// number of keys (when IsOpenSet() == true, as then it gives a number of _excluded_ keys)
	RawKeyCount() int
}

type InclusiveKeySet interface {
	KeySet
	// lists keys
	EnumKeys(func(k Key) bool) bool
	// number of unique keys
	Count() int
}

func New(keys []Key) KeySet {
	n := len(keys)
	switch n {
	case 0:
		return Nothing()
	case 1:
		return SoloKeySet(keys[0])
	}

	ks := make(basicKeySet, n)
	for _, k := range keys {
		ks.add(k)
	}
	return inclusiveKeySet{ks}
}

func Wrap(keys KeyList) KeySet {
	if keys == nil {
		panic("illegal state")
	}
	return inclusiveKeySet{listSet{keys}}
}

func CopyList(keys KeyList) KeySet {
	n := keys.Count()
	if n == 0 {
		return Nothing()
	}
	return inclusiveKeySet{listSet{keys}.copy(0)}
}

func CopySet(keys KeySet) KeySet {
	if iks, ok := keys.(internalKeySet); ok {
		return newKeySet(keys.IsOpenSet(), iks.copy(0))
	}

	switch n := keys.RawKeyCount(); {
	case n > 0:
		ks := make(basicKeySet, n)
		keys.EnumRawKeys(func(k Key, _ bool) bool {
			ks.add(k)
			return false
		})
		return newKeySet(keys.IsOpenSet(), ks)
	case n != 0:
		panic("illegal state")
	case keys.IsOpenSet():
		return Everything()
	default:
		return Nothing()
	}
}

func newKeySet(exclusive bool, ks internalKeySet) KeySet {
	switch {
	case ks == nil:
		panic("illegal value")
	case exclusive:
		return exclusiveKeySet{ks}
	default:
		return inclusiveKeySet{ks}
	}
}
