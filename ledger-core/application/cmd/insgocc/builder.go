package main

import (
	"context"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine/machinetype"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/application/genesisrefs"
	"github.com/insolar/assured-ledger/ledger-core/application/preprocessor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
)

var (
	contractSources = defaults.RootModule + "/application/contract"
	proxySources    = defaults.RootModule + "/application/proxy"
)

type contractsBuilder struct {
	root      string
	skipProxy bool

	sourcesDir string
	outDir     string
}

func (cb *contractsBuilder) setSourcesDir(dir string) {
	cb.sourcesDir = dir
}

func (cb *contractsBuilder) setOutputDir(dir string) {
	cb.outDir = dir
}

func (cb *contractsBuilder) outputDir() string {
	if cb.outDir != "" {
		return cb.outDir
	}
	return filepath.Join(cb.root, "plugins")
}

func newContractBuilder(tmpDir string, skipProxy bool) *contractsBuilder {
	if tmpDir == "" {
		tmpDir = defaults.ContractBuildTmpDir("insgocc-")
	}

	cb := &contractsBuilder{
		root:      tmpDir,
		skipProxy: skipProxy,
	}
	return cb
}

// clean deletes tmp directory used for contracts building
func (cb *contractsBuilder) clean() {
	global.Infof("Cleaning build directory %q", cb.root)
	err := os.RemoveAll(cb.root)
	if err != nil {
		global.Error(err)
	}
}

func (cb *contractsBuilder) parseContract(name string) (*preprocessor.ParsedFile, error) {
	return preprocessor.ParseFile(cb.getContractPath(name), machinetype.Builtin)
}

type buildResult struct {
	ContractName string
	SoFilePath   string
}

func (cb *contractsBuilder) build(ctx context.Context, names ...string) ([]buildResult, error) {
	if err := cb.prepare(ctx, names...); err != nil {
		return nil, err
	}

	result := []buildResult{}
	for _, name := range names {
		global.Infof("building plugin for contract %q in %q", name, cb.root)
		soFile, err := cb.plugin(ctx, name)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build plugin %v", name)
		}
		result = append(result, buildResult{
			ContractName: name,
			SoFilePath:   soFile,
		})
	}

	return result, nil
}

func (cb *contractsBuilder) prepare(ctx context.Context, names ...string) error {
	inslog := inslogger.FromContext(ctx)
	for _, name := range names {
		inslog.Info("prepare contract:", name)
		code, err := cb.parseContract(name)
		if err != nil {
			return errors.Wrapf(err, "failed to parse contract %v", name)
		}

		code.ChangePackageToMain()

		ctr, err := createFileInDir(filepath.Join(cb.root, "src/contract", name), "main.go")
		if err != nil {
			return errors.W(err, "can't create contract file")
		}
		err = code.Write(ctr)
		if err != nil {
			return errors.W(err, "can't write to contract file")
		}
		closeAndCheck(ctr)

		if !cb.skipProxy {
			proxyPath := filepath.Join(cb.root, "src", proxySources, name)
			proxy, err := createFileInDir(proxyPath, "main.go")
			if err != nil {
				return errors.W(err, "can't open proxy file")
			}
			classRef := genesisrefs.GenesisRef(name + genesisrefs.ClassSuffix)
			err = code.WriteProxy(classRef.String(), proxy)
			closeAndCheck(proxy)
			if err != nil {
				return errors.W(err, "can't write proxy")
			}
		}

		wrp, err := createFileInDir(filepath.Join(cb.root, "src/contract", name), "main_wrapper.go")
		if err != nil {
			return errors.W(err, "can't open wrapper file")
		}
		err = code.WriteWrapper(wrp, "main")
		closeAndCheck(wrp)
		if err != nil {
			return errors.W(err, "can't write wrapper")
		}
	}

	return nil
}

// compile plugin
func (cb *contractsBuilder) plugin(ctx context.Context, name string) (string, error) {
	dstDir := cb.outputDir()

	err := os.MkdirAll(dstDir, 0700)
	if err != nil {
		return "", errors.Wrapf(err, "filed to create output directory for plugin %v", dstDir)
	}

	soFile := filepath.Join(dstDir, name+".so")
	buildPath := filepath.Join(cb.root, "src/contract", name)
	args := []string{
		"build",
		"-buildmode=plugin",
		// "-trimpath",
		"-mod=vendor",
		"-o", soFile,
		".",
	}
	cmdVendor := exec.Command("go", "mod", "vendor")
	cmd := exec.Command(
		"go",
		args...,
	)
	cmd.Dir = buildPath
	inslogger.FromContext(ctx).Infof("exec: go %v", strings.Join(args, " "))

	env := make([]string, 0, len(os.Environ()))
	env = append(env, "GO111MODULE=on")
	for _, pair := range os.Environ() {
		if strings.HasPrefix(pair, "GOPATH=") {
			continue
		}
		env = append(env, pair)
	}
	env = append(env, "GOPATH="+prependGoPath(cb.root))
	inslogger.FromContext(ctx).Info("GOPATH=" + prependGoPath(cb.root))
	cmd.Env = env
	cmdVendor.Env = env

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmdVendor.Stdout = os.Stdout
	cmdVendor.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "can't build plugin: %v", soFile)
	}
	inslogger.FromContext(ctx).Infof("compiled %v contract to plugin %v", name, soFile)
	return soFile, nil
}

func goPATH() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	return gopath
}

func (cb *contractsBuilder) getContractPath(name string) string {
	contractDir := filepath.Join(goPATH(), "src", contractSources)
	if cb.sourcesDir != "" {
		contractDir = cb.sourcesDir
	}
	contractFile := name + ".go"
	return filepath.Join(contractDir, name, contractFile)
}

// prependGoPath prepends `path` to GOPATH environment variable
// accounting for possibly for default value. Returns new value.
// NOTE: that environment is not changed
func prependGoPath(path string) string {
	return path + string(os.PathListSeparator) + goPATH()
}

// createFileInDir opens file in provided directory, creates directory if it does not exist.
func createFileInDir(dir string, name string) (*os.File, error) {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
}

func closeAndCheck(f *os.File) {
	err := f.Close()
	if err != nil {
		global.Errorf("failed close file %v: %v", f.Name(), err.Error())
	}
}
