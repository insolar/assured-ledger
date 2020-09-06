// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package configuration

import (
	"crypto"
	"fmt"

	"github.com/insolar/insconfig"
	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
)

const InsolarEnvPrefix = "insolar"

// общий конфиг для Net и App
// в режиме full-node --single-process запускается один инстанс компонентов
// проблема идентичности CommonConfig в режиме full-node --independent.
// Решили дублировать эту секцию для Net и App, в режиме --single-process берется CommonConfig из секции Network
type CommonConfig struct {
	Log           Log
	Metrics       Metrics
	Tracer        Tracer
	Introspection Introspection
	KeysPath      string
}

type NetworkConfig struct {
	CommonConfig
	CertificatePath string
	Host            HostNetwork
	Service         ServiceNetwork
}

type CommonAppConfig struct {
	CommonConfig
}

type VirtualNodeConfig struct {
	CommonAppConfig
	LogicRunner LogicRunner
}

type LightNodeConfig struct {
	CommonAppConfig
	LogicRunner LogicRunner
}

type HeavyNodeConfig struct {
	CommonAppConfig
	LogicRunner LogicRunner
}

type BaseCloudConfig struct {
	Log                 Log
	PulsarConfiguration PulsarConfiguration
	NodeConfigPaths     []string
}

type CertManagerFactory func(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*mandates.CertificateManager, error)
type KeyStoreFactory func(path string) (cryptography.KeyStore, error)

type CloudConfigurationProvider struct {
	CloudConfig        BaseCloudConfig
	CertificateFactory CertManagerFactory
	KeyFactory         KeyStoreFactory
	GetAppConfigs      func() []Configuration
}

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
	TestWalletAPI       TestWalletAPI
	AvailabilityChecker AvailabilityChecker
	KeysPath            string
	CertificatePath     string
	Tracer              Tracer
	Introspection       Introspection
}

// PulsarConfiguration contains configuration params for the pulsar node
type PulsarConfiguration struct {
	Log      Log
	Pulsar   Pulsar
	Tracer   Tracer
	KeysPath string
	Metrics  Metrics
	OneShot  bool
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
		OneShot:  false,
	}
}

// NewHolder creates new Holder with default configuration
func NewHolder(path string) *Holder {
	params := insconfig.Params{
		EnvPrefix:        InsolarEnvPrefix,
		ConfigPathGetter: &StringPathGetter{path},
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

type StringPathGetter struct {
	Path string
}

func (g *StringPathGetter) GetConfigPath() string {
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
