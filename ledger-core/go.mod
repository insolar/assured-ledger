module github.com/insolar/assured-ledger/ledger-core

go 1.14

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/ThreeDotsLabs/watermill v1.0.2
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/cyraxred/go-acc v0.2.5
	github.com/fsnotify/fsnotify v1.4.9
	github.com/getkin/kin-openapi v0.2.1-0.20191211203508-0d9caf80ada6
	github.com/gogo/protobuf v1.3.1
	github.com/gojuno/minimock/v3 v3.0.6
	github.com/golang/protobuf v1.4.0
	github.com/google/gofuzz v1.0.0
	github.com/google/gops v0.3.11
	github.com/grpc-ecosystem/grpc-gateway v1.9.6
	github.com/insolar/component-manager v0.2.1-0.20191028200619-751a91771d2f
	github.com/insolar/consensus-reports v0.0.0-20200515131339-fea7a784f1d6 // indirect
	github.com/insolar/gls v0.0.0-20200427111849-9a08a622625d
	github.com/insolar/insconfig v0.0.0-20200513150834-977022bc1445
	github.com/insolar/rpc v1.2.2-0.20190812143745-c27e1d218f1f
	github.com/insolar/sm-uml-gen v1.0.0
	github.com/insolar/x-crypto v0.0.0-20191031140942-75fab8a325f6
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/json-iterator/go v1.1.9
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/neilotoole/errgroup v0.1.5
	github.com/olekukonko/tablewriter v0.0.1
	github.com/opentracing/opentracing-go v1.1.0
	github.com/paulbellamy/ratecounter v0.2.0
	github.com/prometheus/client_golang v1.6.0
	github.com/rs/zerolog v1.15.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/spf13/afero v1.3.2 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1
	github.com/tylerb/is v2.1.4+incompatible // indirect
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.19.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8
	go.opencensus.io v0.22.1
	go.uber.org/goleak v1.0.0
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sys v0.0.0-20200824131525-c12d262b63d8
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/tools v0.0.0-20200312153518-5e2df02acb1e
	google.golang.org/genproto v0.0.0-20191108220845-16a3f7862a1a
	google.golang.org/grpc v1.22.0
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
)

replace github.com/insolar/assured-ledger/ledger-core => ./

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
