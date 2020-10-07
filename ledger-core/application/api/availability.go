// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/application/api.AvailabilityChecker -s _mock.go -g

// AvailabilityChecker component checks if insolar network can't process any new requests
type AvailabilityChecker interface {
	IsAvailable(context.Context) bool
}

type keeperResponse struct {
	Available bool `json:"available"`
}

// NetworkChecker is AvailabilityChecker implementation that checks can we process any API requests based on keeper status
type NetworkChecker struct {
	client      *http.Client
	enabled     bool
	keeperURL   string
	checkPeriod time.Duration
	stopped     chan struct{}

	lock        *sync.RWMutex
	isAvailable bool
}

func NewNetworkChecker(cfg configuration.AvailabilityChecker) *NetworkChecker {
	return &NetworkChecker{
		client: &http.Client{
			Transport: &http.Transport{},
			Timeout:   time.Duration(cfg.RequestTimeout) * time.Second,
		},

		enabled:     cfg.Enabled,
		keeperURL:   cfg.KeeperURL,
		checkPeriod: time.Duration(cfg.CheckPeriod) * time.Second,
		stopped:     make(chan struct{}),
		lock:        &sync.RWMutex{},
		isAvailable: false,
	}
}

func (nc *NetworkChecker) Start(ctx context.Context) error {
	if !nc.enabled {
		nc.lock.Lock()
		defer nc.lock.Unlock()

		nc.isAvailable = true
		return nil
	}

	go func(ctx context.Context) {
		ticker := time.NewTicker(nc.checkPeriod)
		defer ticker.Stop()

		stop := false
		for !stop {
			select {
			case <-ticker.C:
				nc.UpdateAvailability(ctx)
			case <-nc.stopped:
				stop = true
			}
		}
		nc.stopped <- struct{}{}
	}(ctx)
	return nil
}

func (nc *NetworkChecker) Stop() {
	nc.stopped <- struct{}{}
	<-nc.stopped
}

func (nc *NetworkChecker) UpdateAvailability(ctx context.Context) {
	logger := inslogger.FromContext(ctx)
	logger.Debug("[ NetworkChecker ] update availability started")
	resp, err := nc.client.Get(nc.keeperURL)
	defer func() {
		if resp != nil && resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				logger.Error("[ NetworkChecker ] Can't close body: ", err)
			}
		}
	}()

	nc.lock.Lock()
	defer nc.lock.Unlock()

	if err != nil {
		nc.isAvailable = false
		logger.Error("[ NetworkChecker ] Can't get keeper status: ", err)
		return
	}

	if resp == nil || resp.StatusCode != http.StatusOK {
		nc.isAvailable = false
		logger.Error("[ NetworkChecker ] Can't get keeper status: no response or bad StatusCode: ", resp.StatusCode)
		return
	}

	respObj := &keeperResponse{}
	err = json.NewDecoder(resp.Body).Decode(respObj)
	if err != nil {
		nc.isAvailable = false
		logger.Error("[ NetworkChecker ] Can't get keeper status: Can't decode body: ", err)
		return
	}

	if !respObj.Available {
		logger.Warn("[ NetworkChecker ] Network is not available for request processing")
	}
	nc.isAvailable = respObj.Available
}

func (nc *NetworkChecker) IsAvailable(ctx context.Context) bool {
	nc.lock.RLock()
	defer nc.lock.RUnlock()
	return nc.isAvailable
}

type DummyNetworkChecker struct{}

func NewDummyNetworkChecker(_ configuration.AvailabilityChecker) *DummyNetworkChecker {
	return &DummyNetworkChecker{}
}

func (d DummyNetworkChecker) IsAvailable(ctx context.Context) bool { return true }
