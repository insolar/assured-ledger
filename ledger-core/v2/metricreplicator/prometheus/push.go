package prometheus

import (
	"context"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/studio-b12/gowebdav"

	"github.com/insolar/assured-ledger/ledger-core/v2/metricreplicator/replicator"
)

func (repl Replicator) PushFiles(ctx context.Context, cfg replicator.ReplicaConfig, files []string) error {
	client := gowebdav.NewClient(cfg.URL, cfg.User, cfg.Password)

	if err := client.Mkdir(cfg.RemoteDirName, 0644); err != nil {
		return errors.Wrap(err, "failed to create remote dir")
	}

	for _, f := range files {
		localFilePath := repl.TmpDir + "/" + f
		data, err := ioutil.ReadFile(localFilePath)
		if err != nil {
			return errors.Wrap(err, "failed to read local file")
		}

		remoteFilePath := cfg.RemoteDirName + "/" + f
		if err := client.Write(remoteFilePath, data, 0644); err != nil {
			return errors.Wrap(err, "failed to write data to remote file")
		}
	}
	return nil
}
