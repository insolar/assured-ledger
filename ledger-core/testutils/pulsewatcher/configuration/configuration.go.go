// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package configuration

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
)

type OutputFormat string

const (
	Txt  OutputFormat = "text"
	Json OutputFormat = "json"
)

type Config struct {
	Nodes     []string
	Interval  time.Duration
	Timeout   time.Duration
	Log       configuration.Log
	Format    OutputFormat
	ShowEmoji bool
	OneShot   bool
}

func NewPulseWatcherConfiguration() Config {
	return Config{
		Interval: 500 * time.Millisecond,
		Timeout:  1 * time.Second,
		Log:      configuration.NewLog(),
		Format:   Txt,
	}
}
