module github.com/insolar/assured-ledger/ledger-core

go 1.14

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/ThreeDotsLabs/watermill v1.0.2
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/fsnotify/fsnotify v1.4.7
	github.com/getkin/kin-openapi v0.2.1-0.20191211203508-0d9caf80ada6
	github.com/gogo/protobuf v1.3.1
	github.com/gojuno/minimock/v3 v3.0.6
	github.com/golang/protobuf v1.3.2
	github.com/google/gofuzz v1.0.0
	github.com/google/gops v0.3.6
	github.com/grpc-ecosystem/grpc-gateway v1.9.6
	github.com/insolar/component-manager v0.2.1-0.20191028200619-751a91771d2f
	github.com/insolar/gls v0.0.0-20200427111849-9a08a622625d
	github.com/insolar/insconfig v0.0.0-20200228110347-69b2648d7227
	github.com/insolar/rpc v1.2.2-0.20190812143745-c27e1d218f1f
	github.com/insolar/sm-uml-gen v0.0.0-20200613174513-58a1c59e24de
	github.com/insolar/x-crypto v0.0.0-20191031140942-75fab8a325f6
	github.com/jbenet/go-base58 v0.0.0-20150317085156-6237cf65f3a6
	github.com/json-iterator/go v1.1.6
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onrik/gomerkle v1.0.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/ory/go-acc v0.2.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.4 // indirect
	github.com/rs/zerolog v1.15.0
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	github.com/tylerb/is v2.1.4+incompatible // indirect
	github.com/uber-go/atomic v1.4.0 // indirect
	github.com/uber/jaeger-client-go v2.19.0+incompatible
	github.com/uber/jaeger-lib v2.2.0+incompatible // indirect
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8
	go.opencensus.io v0.22.1
	go.uber.org/goleak v1.0.0
	golang.org/x/crypto v0.0.0-20191206172530-e9b2fee46413
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd
	golang.org/x/tools v0.0.0-20200312153518-5e2df02acb1e
	google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/grpc v1.22.0
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	gotest.tools/gotestsum v0.4.1
)

replace github.com/insolar/assured-ledger/ledger-core => ./

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
