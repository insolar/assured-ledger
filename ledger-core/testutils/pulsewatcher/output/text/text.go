// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package text

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/testutils/pulsewatcher/configuration"
	"github.com/insolar/assured-ledger/ledger-core/testutils/pulsewatcher/output"
	"github.com/insolar/assured-ledger/ledger-core/testutils/pulsewatcher/status"
)

const (
	esc       = "\x1b%s"
	moveUp    = "[%dA"
	clearDown = "[0J"

	insolarReady    = "Ready"
	insolarNotReady = "Not Ready"
)

func escape(format string, args ...interface{}) string {
	return fmt.Sprintf(esc, fmt.Sprintf(format, args...))
}

func moveBack(reader io.Reader) {
	fileScanner := bufio.NewScanner(reader)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}

	fmt.Print(escape(moveUp, lineCount))
	fmt.Print(escape(clearDown))
}

type Output struct {
	emoji   Emojer
	buffer  *bytes.Buffer
	started time.Time
}

func NewOutput(cfg configuration.Config) output.Printer {
	var emoji Emojer
	if cfg.ShowEmoji {
		emoji = NewEmoji()
	} else {
		emoji = &NoEmoji{}
	}

	return &Output{
		emoji:   emoji,
		buffer:  bytes.NewBuffer(nil),
		started: time.Now(),
	}
}

func (o *Output) Print(results []status.Node) {
	table := tablewriter.NewWriter(o.buffer)
	table.SetHeader([]string{
		"URL",
		"State",
		"ID",
		"Network Pulse",
		"Pulse",
		"Active",
		"Working",
		"Role",
		"Timestamp",
		"Uptime",
		"Error",
	})
	table.SetBorder(false)

	table.ClearRows()
	table.ClearFooter()

	moveBack(o.buffer)
	o.buffer.Reset()

	ready := true
	for _, node := range results {
		ready = ready && node.Reply.NetworkState == network.CompleteNetworkState.String()
	}

	stateString := insolarReady
	color := tablewriter.FgHiGreenColor
	if !ready {
		stateString = insolarNotReady
		color = tablewriter.FgHiRedColor
	}

	table.SetFooter([]string{
		"", "", "", "",
		"Insolar State", stateString,
		"Time", time.Now().Format(output.TimeFormat),
		"Insolar Uptime", time.Since(o.started).Round(time.Second).String(), "",
	})
	table.SetFooterColor(
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},

		tablewriter.Colors{color},
		tablewriter.Colors{},

		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
	)
	table.SetColumnColor(
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},

		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{},
		tablewriter.Colors{tablewriter.FgHiRedColor},
	)

	intToString := func(n int) string {
		if n == 0 {
			return ""
		}
		return strconv.Itoa(n)
	}

	shortRole := func(r string) string {
		switch r {
		case "virtual":
			return "Virtual"
		case "heavy_material":
			return "Heavy"
		case "light_material":
			return "Light"
		default:
			return r
		}
	}

	for _, row := range results {
		o.emoji.RegisterNode(row.URL, row.Reply.Origin)
	}

	for _, row := range results {
		var activeNodeEmoji string
		for _, n := range row.Reply.Nodes {
			activeNodeEmoji += o.emoji.GetEmoji(n)
		}

		var uptime string
		var timestamp string
		if row.ErrStr == "" {
			uptime = time.Since(row.Reply.StartTime).Round(time.Second).String()
			timestamp = row.Reply.Timestamp.Format(output.TimeFormat)
		}

		table.Append([]string{
			row.URL,
			row.Reply.NetworkState,
			fmt.Sprintf(" %s %s", o.emoji.GetEmoji(row.Reply.Origin), intToString(int(row.Reply.Origin.ID))),
			intToString(int(row.Reply.NetworkPulseNumber)),
			intToString(int(row.Reply.PulseNumber)),
			fmt.Sprintf("%d %s", row.Reply.ActiveListSize, activeNodeEmoji),
			intToString(row.Reply.WorkingListSize),
			shortRole(row.Reply.Origin.Role),
			timestamp,
			uptime,
			row.ErrStr,
		})
	}
	table.Render()
	fmt.Print(o.buffer)
}
