package prometheus

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

type Replicator struct {
	Address   string
	TmpDir    string
	Commit    string
	APIClient v1.API
}

func NewReplicator(address, commit, tmpDir string) (replicator.Replicator, error) {
	repl := Replicator{
		Address: address,
		Commit:  commit,
		TmpDir:  tmpDir,
	}

	client, err := api.NewClient(api.Config{Address: repl.Address})
	if err != nil {
		return Replicator{}, errors.Wrap(err, "failed to create prometheus client")
	}

	repl.APIClient = v1.NewAPI(client)
	return repl, nil
}

func (repl Replicator) ToTmpDir() (replicator.Clean, error) {
	if err := os.Mkdir(repl.TmpDir, 0777); err != nil {
		return func() {}, errors.Wrap(err, "failed to create tmp dir")
	}

	removeFunc := func() {
		if err := os.RemoveAll(repl.TmpDir); err != nil {
			log.Printf("failed to remove tmp dir: %v", err)
		}
	}

	if err := os.Chdir(repl.TmpDir); err != nil {
		return removeFunc, errors.Wrap(err, "cant change dir to tmp directory")
	}
	return removeFunc, nil
}

func (repl Replicator) saveDataToFile(data []byte, filename string) error {
	if _, err := os.Stat(filename); err == nil {
		return errors.Errorf("file already exists: %s", filename)
	}

	recordFile, createErr := os.Create(filename)
	if createErr != nil {
		return errors.Wrap(createErr, "failed to create file")
	}
	defer recordFile.Close()

	_, writeErr := recordFile.Write(data)
	if writeErr != nil {
		return errors.Wrap(writeErr, "failed to write file")
	}
	return nil
}

const indexFile = "index.json"

func (repl Replicator) MakeIndexFile(ctx context.Context, files []string) (string, error) {
	indexData, err := json.Marshal(files)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal files list")
	}

	if err := repl.saveDataToFile(indexData, indexFile); err != nil {
		return "", errors.Wrap(err, "failed to save index file")
	}
	return indexFile, nil
}
