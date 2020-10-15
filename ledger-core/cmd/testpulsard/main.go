// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/pulsenetwork"
	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/version"
)

type inputParams struct {
	configPath string
	port       string
}

func parseInputParams() inputParams {
	var rootCmd = &cobra.Command{Use: "insolard"}
	var result inputParams
	rootCmd.Flags().StringVarP(&result.configPath, "config", "c", "", "path to config file")
	rootCmd.Flags().StringVarP(&result.port, "port", "port", "", "port for test pulsar")
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("Wrong input params:", err.Error())
	}

	return result
}

func main() {
	params := parseInputParams()

	vp := viper.New()
	pCfg := configuration.NewPulsarConfiguration()
	if len(params.configPath) != 0 {
		vp.SetConfigFile(params.configPath)
	}
	err := vp.ReadInConfig()
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}
	err = vp.Unmarshal(&pCfg)
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}

	ctx := context.Background()
	ctx, _ = inslogger.InitGlobalNodeLogger(ctx, pCfg.Log, "", "test_pulsar")
	testPulsar := initPulsar(ctx, pCfg)

	http.HandleFunc("/pulse", func(writer http.ResponseWriter, request *http.Request) {
		err := testPulsar.SendPulse(ctx)
		if err != nil {
			_, err := fmt.Fprintf(writer, "Error - %v", err)
			if err != nil {
				panic(err)
			}
		}

		_, err = fmt.Fprint(writer, "OK")
		if err != nil {
			panic(err)
		}
	})

	fmt.Printf("Starting server for testing HTTP POST...\n")
	if err := http.ListenAndServe(params.port, nil); err != nil {
		panic(err)
	}
}

func initPulsar(ctx context.Context, cfg configuration.PulsarConfiguration) *pulsar.TestPulsar {
	fmt.Println("Version: ", version.GetFullVersion())
	fmt.Println("Starts with configuration:\n", configuration.ToString(cfg))

	keyStore, err := keystore.NewKeyStore(cfg.KeysPath)
	if err != nil {
		panic(err)
	}
	cryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	cryptographyService := platformpolicy.NewCryptographyService()
	keyProcessor := platformpolicy.NewKeyProcessor()

	pulseDistributor, err := pulsenetwork.NewDistributor(cfg.Pulsar.PulseDistributor, pulsenetwork.NewPulsarUniserver())
	if err != nil {
		panic(err)
	}

	cm := component.NewManager(nil)
	cm.SetLogger(global.Logger())

	cm.Register(cryptographyScheme, keyStore, keyProcessor)
	cm.Inject(cryptographyService, pulseDistributor)

	if err = cm.Init(ctx); err != nil {
		panic(err)
	}

	if err = cm.Start(ctx); err != nil {
		panic(err)
	}

	return pulsar.NewTestPulsar(cfg.Pulsar, pulseDistributor, &entropygenerator.StandardEntropyGenerator{})
}
