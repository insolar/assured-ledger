package defaults

import (
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// RootModule holds root module name.
var RootModule = "github.com/insolar/assured-ledger/ledger-core"

// RootModuleDir returns abs path to root module for any package where it's called.
func RootModuleDir() string {
	p, err := build.Default.Import(RootModule, ".", build.FindOnly)
	if err != nil {
		log.Fatal("failed to resolve", RootModule)
	}
	return p.Dir
}

func ContractBuildTmpDir(prefix string) string {
	dir := filepath.Join(RootModuleDir(), ArtifactsDir(), "tmp")
	// create if not exist
	if err := os.MkdirAll(dir, 0777); err != nil {
		panic(err)
	}

	tmpDir, err := ioutil.TempDir(dir, prefix)
	if err != nil {
		panic(err)
	}
	return tmpDir
}
