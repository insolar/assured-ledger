// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package configuration

import (
	"fmt"
	"github.com/insolar/insconfig"

	yaml "gopkg.in/yaml.v2"
)

const InsolarEnvPrefix = "insolar"

// Configuration contains configuration params for all Insolar components
type Configuration struct {
	Host                HostNetwork
	Service             ServiceNetwork
	Ledger              Ledger
	Log                 Log
	Metrics             Metrics
	LogicRunner         LogicRunner
	APIRunner           APIRunner
	AdminAPIRunner      APIRunner
	AvailabilityChecker AvailabilityChecker
	KeysPath            string
	CertificatePath     string
	Tracer              Tracer
	Introspection       Introspection
	Exporter            Exporter
	Bus                 Bus
}

// PulsarConfiguration contains configuration params for the pulsar node
type PulsarConfiguration struct {
	Log      Log
	Pulsar   Pulsar
	Tracer   Tracer
	KeysPath string
	Metrics  Metrics
}

// Holder provides methods to manage configuration
type Holder struct {
	Configuration *Configuration
	Params        insconfig.Params
}

// NewConfiguration creates new default configuration
func NewConfiguration() Configuration {
	cfg := Configuration{
		Host:                NewHostNetwork(),
		Service:             NewServiceNetwork(),
		Ledger:              NewLedger(),
		Log:                 NewLog(),
		Metrics:             NewMetrics(),
		LogicRunner:         NewLogicRunner(),
		APIRunner:           NewAPIRunner(false),
		AdminAPIRunner:      NewAPIRunner(true),
		AvailabilityChecker: NewAvailabilityChecker(),
		KeysPath:            "./",
		CertificatePath:     "",
		Tracer:              NewTracer(),
		Introspection:       NewIntrospection(),
		Exporter:            NewExporter(),
		Bus:                 NewBus(),
	}

	return cfg
}

// NewPulsarConfiguration creates new default configuration for pulsar
func NewPulsarConfiguration() PulsarConfiguration {
	return PulsarConfiguration{
		Log:      NewLog(),
		Pulsar:   NewPulsar(),
		Tracer:   NewTracer(),
		KeysPath: "./",
		Metrics:  NewMetrics(),
	}
}

// NewHolder creates new Holder with default configuration
func NewHolder(path string) *Holder {
	params := insconfig.Params{
		EnvPrefix:        InsolarEnvPrefix,
		ConfigPathGetter: &stringPathGetter{path},
	}

	return &Holder{&Configuration{}, params}
}

// Load method reads configuration from params file path
func (h *Holder) Load() error {
	insConfigurator := insconfig.New(h.Params)
	if err := insConfigurator.Load(h.Configuration); err != nil {
		return err
	}
	return nil
}

// MustLoad wrapper around Load method with panics on error
func (h *Holder) MustLoad() *Holder {
	err := h.Load()
	if err != nil {
		panic(err)
	}
	return h
}

type stringPathGetter struct {
	Path string
}

func (g *stringPathGetter) GetConfigPath() string {
	return g.Path
}

// ToString converts any configuration struct to yaml string
func ToString(in interface{}) string {
	d, err := yaml.Marshal(in)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(d)
}
