package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

type RecordInfo struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

type ResultData struct {
	Warnings    []string     `json:"warnings"`
	Records     []RecordInfo `json:"records"`
	NetworkSize uint         `json:"network_size"`
}

func (repl Replicator) grabQueryRangeValue(ctx context.Context, query string, start, end time.Time) (ResultData, error) {
	var recordData ResultData

	queryCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	r := v1.Range{
		Start: start,
		End:   end,
		Step:  end.Sub(start),
	}
	result, warnings, queryErr := repl.APIClient.QueryRange(queryCtx, query, r)
	if queryErr != nil {
		return ResultData{}, errors.Wrap(queryErr, "failed to query prometheus")
	}

	recordData.Warnings = warnings
	// Unless it will be {"warnings":null} in json
	if len(recordData.Warnings) == 0 {
		recordData.Warnings = make([]string, 0)
	}

	var parseErr error
	switch result.Type() {
	case model.ValMatrix:
		recordData.Records, parseErr = quantileRecordsFromMatrix(result, query)
	default:
		return ResultData{}, errors.Errorf("unknown type of result: %s, can't parse", result.Type().String())
	}
	if parseErr != nil {
		return ResultData{}, errors.Wrap(parseErr, "failed to parse result")
	}

	return recordData, nil
}

func quantileRecordsFromMatrix(result model.Value, metric string) ([]RecordInfo, error) {
	matrixResult, ok := result.(model.Matrix)
	if !ok {
		return []RecordInfo{}, errors.Errorf("failed to cast result type to %T", model.Matrix{})
	}

	if len(matrixResult) != 1 || len(matrixResult[0].Values) != 2 {
		return []RecordInfo{}, errors.Errorf("wrong number of records: %v", matrixResult)
	}

	record := RecordInfo{
		Metric: metric,
		Value:  float64(matrixResult[0].Values[1].Value - matrixResult[0].Values[0].Value),
	}
	return []RecordInfo{record}, nil
}

const (
	quantileQuery = "quantile(%s, sum by (instance) (%s))"
)

func (repl Replicator) grabQuantileMetric(ctx context.Context, params replicator.QuantileParams) (json.RawMessage, error) {
	response := make([]ResultData, 0)
	for _, r := range params.Ranges {
		for _, quantile := range params.Quantiles {
			query := fmt.Sprintf(quantileQuery, quantile, params.Metric)

			recordData, grabErr := repl.grabQueryRangeValue(ctx, query, r.Start, r.End)
			if grabErr != nil {
				return nil, errors.Wrap(grabErr, "failed to get result for query")
			}

			recordData.NetworkSize = r.NetworkSize
			response = append(response, recordData)
		}
	}

	rawMsg, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return nil, errors.Wrap(marshalErr, "failed to marshal response")
	}
	return rawMsg, nil
}

func (repl Replicator) grabLatencyMetric(ctx context.Context, params replicator.LatencyParams) (json.RawMessage, error) {
	response := make([]ResultData, 0)
	for _, r := range params.Ranges {
		query := "todo"

		recordData, grabErr := repl.grabQueryRangeValue(ctx, query, r.Start, r.End)
		if grabErr != nil {
			return nil, errors.Wrap(grabErr, "failed to get result for query")
		}

		recordData.NetworkSize = r.NetworkSize
		response = append(response, recordData)
	}

	rawMsg, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return nil, errors.Wrap(marshalErr, "failed to marshal response")
	}
	return rawMsg, nil
}

func (repl Replicator) GrabQuantileRecords(ctx context.Context, params []replicator.QuantileParams) ([]string, error) {
	files := make([]string, 0, len(params))

	for _, p := range params {
		data, grabErr := repl.grabQuantileMetric(ctx, p)
		if grabErr != nil {
			return []string{}, errors.Wrap(grabErr, "failed to grab records")
		}

		filename := fmt.Sprintf(p.FileName, p.Metric)
		if err := repl.saveDataToFile(data, filename); err != nil {
			return []string{}, errors.Wrap(err, "failed to save records")
		}

		files = append(files, filename)
	}
	return files, nil
}

func (repl Replicator) GrabLatencyRecords(ctx context.Context, params []replicator.LatencyParams) ([]string, error) {
	files := make([]string, 0, len(params))

	for _, p := range params {
		data, grabErr := repl.grabLatencyMetric(ctx, p)
		if grabErr != nil {
			return []string{}, errors.Wrap(grabErr, "failed to grab records")
		}

		filename := fmt.Sprintf(p.FileName, p.Metric)
		if err := repl.saveDataToFile(data, filename); err != nil {
			return []string{}, errors.Wrap(err, "failed to save records")
		}

		files = append(files, filename)
	}
	return files, nil
}
