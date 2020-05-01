// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"net/rpc"
	"os"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/global"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/goplugin/rpctypes"
)

func main() {
	rpcAddress := pflag.StringP("rpc", "a", "", "address and port of RPC API")
	rpcProtocol := pflag.StringP("rpc-proto", "p", "tcp", "protocol of RPC API, tcp by default")
	refString := pflag.StringP("ref", "r", "", "ref of healthcheck contract")
	pflag.Parse()

	if *rpcAddress == "" || *rpcProtocol == "" || *refString == "" {
		global.Error(errors.New("need to provide all params"))
		os.Exit(2)
	}

	client, err := rpc.Dial(*rpcProtocol, *rpcAddress)
	if err != nil {
		global.Error(err.Error())
		os.Exit(2)
	}

	ref, err := reference.GlobalFromString(*refString)
	if err != nil {
		global.Errorf("Failed to parse healthcheck contract ref: %s", err.Error())
		os.Exit(2)
	}

	empty, _ := insolar.Serialize([]interface{}{})

	caller := gen.Reference()
	res := rpctypes.DownCallMethodResp{}
	req := rpctypes.DownCallMethodReq{
		Context:   &insolar.LogicCallContext{Caller: &caller},
		Code:      ref,
		Data:      empty,
		Method:    "Check",
		Arguments: empty,
	}

	err = client.Call("RPC.CallMethod", req, &res)
	if err != nil {
		global.Error(err.Error())
		os.Exit(2)
	}
}
