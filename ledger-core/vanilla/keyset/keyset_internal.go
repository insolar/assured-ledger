package keyset

type internalKeySet interface {
	KeyList
	enumRawKeys(exclusive bool, fn func(k Key, exclusive bool) bool) bool
	copy(n int) basicKeySet
}

// read-only access only
type copyKeySet interface {
	KeySet
	copy(n int) basicKeySet
}

// mutable access
type mutableKeySet interface {
	copyKeySet
	retainAll(ks KeySet) mutableKeySet
	removeAll(ks KeySet) mutableKeySet
	addAll(ks KeySet) mutableKeySet
	removeKeys(k []Key)
	addKeys(k []Key)
	remove(k Key)
	add(k Key)
}

var _ internalKeySet = listSet{}

type listSet struct {
	KeyList
}

func (v listSet) enumRawKeys(exclusive bool, fn func(k Key, exclusive bool) bool) bool {
	return v.KeyList.EnumKeys(func(k Key) bool {
		return fn(k, exclusive)
	})
}

func (v listSet) copy(n int) basicKeySet {
	if nn := v.Count(); n < nn {
		n = nn
	}
	if n == 0 {
		return nil
	}
	r := make(basicKeySet, n)
	v.KeyList.EnumKeys(func(k Key) bool {
		r.add(k)
		return false
	})
	return r
}

func (v basicKeySet) copy(n int) basicKeySet {
	if nn := len(v); n < nn {
		n = nn
	}
	if n == 0 {
		return nil
	}
	r := make(basicKeySet, n)
	for k := range v {
		r[k] = struct{}{}
	}
	return r
}
