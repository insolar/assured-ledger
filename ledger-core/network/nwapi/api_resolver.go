package nwapi

import (
	"context"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi/nwaddr"
)

type BasicAddressResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

type AddressResolver interface {
	BasicAddressResolver
	LookupNetworkAddress(ctx context.Context, a Address) ([]net.IPAddr, error)
}

type resolverFunc = func(ctx context.Context, address Address) ([]Address, error)

func newAddResolver(resolver *net.Resolver) resolverFunc {
	return func(ctx context.Context, address Address) ([]Address, error) {
		if !address.IsNetCompatible() {
			return nil, throw.Unsupported()
		}
		list, err := resolver.LookupIPAddr(ctx, address.HostString())
		if err != nil {
			return nil, err
		}
		return newAddresses(list, address.port), nil
	}
}

func ExpandHostAddresses(ctx context.Context, skipError bool, resolver *net.Resolver, a ...Address) ([]Address, int, error) {
	return ExpandAddresses(ctx, skipError, newAddResolver(resolver), a...)
}

func ResolveHostAddresses(ctx context.Context, skipError bool, resolver *net.Resolver, a ...Address) ([]Address, error) {
	return ResolveAddresses(ctx, skipError, newAddResolver(resolver), a...)
}

func newAddresses(a []net.IPAddr, port uint16) []Address {
	n := len(a)
	if n == 0 {
		return nil
	}
	result := make([]Address, n)
	for i, addr := range a {
		result[i] = NewIP(addr)
		result[i].port = port
	}
	return result
}

func ExpandAddresses(ctx context.Context, skipError bool, resolverFn func(context.Context, Address) ([]Address, error), a ...Address) (result []Address, resolved int, err error) {
	if len(a) == 0 {
		return nil, 0, nil
	}

	result = make([]Address, 0, len(a)+1)
	keep := make([]bool, len(a))
	for i, aa := range a {
		list, e := resolverFn(ctx, aa)
		switch {
		case e == nil || err != nil:
			//
		case skipError:
			err = e
		default:
			return nil, 0, e
		}
		result = append(result, list...)
		switch len(list) {
		case 0:
			continue
		case 1:
			keep[i] = aa.IdentityType() != nwaddr.IP
		default:
			keep[i] = true
		}
	}
	resolved = len(result)
	for i, ok := range keep {
		if ok {
			result = append(result, a[i])
		}
	}

	return result, resolved, err
}

func ResolveAddresses(ctx context.Context, skipError bool, resolverFn func(context.Context, Address) ([]Address, error), a ...Address) (result []Address, err error) {
	if len(a) == 0 {
		return nil, nil
	}

	result = make([]Address, 0, len(a)+1)
	for i := range a {
		list, e := resolverFn(ctx, a[i])
		switch {
		case e == nil || err != nil:
			//
		case skipError:
			err = e
		default:
			return nil, e
		}
		result = append(result, list...)
	}
	return result, err
}
