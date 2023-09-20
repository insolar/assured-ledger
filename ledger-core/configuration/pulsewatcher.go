package configuration

import (
	"time"
)

type PulseWatcherOutputFormat string

const (
	PulseWatcherOutputTxt  PulseWatcherOutputFormat = "text"
	PulseWatcherOutputJSON PulseWatcherOutputFormat = "json"
)

type PulseWatcherConfig struct {
	Nodes     []string
	Interval  time.Duration
	Timeout   time.Duration
	Log       Log
	Format    PulseWatcherOutputFormat
	ShowEmoji bool
	OneShot   bool
}

func NewPulseWatcherConfiguration() PulseWatcherConfig {
	return PulseWatcherConfig{
		Interval: 500 * time.Millisecond,
		Timeout:  1 * time.Second,
		Log:      NewLog(),
		Format:   PulseWatcherOutputTxt,
	}
}
