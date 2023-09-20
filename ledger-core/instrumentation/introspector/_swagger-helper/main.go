// swagger helper generator:
// encodes <name>.swagger.json files to literals with name <name>Swagger in swagger_const_gen.go file.

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/log/global"
)

var (
	pkgName     = "introspector"
	outFileName = "swagger_const_gen.go"
	suffix      = ".swagger.json"
)

var inDir = flag.String("in", ".", "directory with swagger files")

// copyFile copies file `fromName` to file `toName`. Sadly there is no such method in standard library.
func copyFile(fromName, toName string) error {
	from, err := os.Open(fromName)
	if err != nil {
		return errors.W(err, "os.Open")
	}
	defer from.Close() // nolint

	// We can't use OpenFile here. File should be truncated if it existed.
	to, err := os.Create(toName)
	if err != nil {
		return errors.W(err, "os.Create")
	}
	defer to.Close() // nolint

	_, err = io.Copy(to, from)
	if err != nil {
		return errors.W(err, "io.Copy")
	}
	return nil
}

func main() {
	flag.Parse()

	tmpF, err := ioutil.TempFile("", "swghelper_*.go")
	if err != nil {
		global.Fatal("failed open tmp file:", err)
	}

	files, err := ioutil.ReadDir(*inDir)
	if err != nil {
		global.Fatal("filed to read current directory", err)
	}

	sw := &strictWriter{w: tmpF}
	sw.writeString(fmt.Sprintf("package %v\n", pkgName))
	sw.writeString(preambula)
	sw.writeString("const (\n")
	for _, info := range files {
		if strings.HasSuffix(info.Name(), suffix) {
			name := strings.TrimSuffix(info.Name(), suffix)
			sw.writeString(name + "Swagger = `")
			filePath := path.Join(*inDir, info.Name())
			f, err := os.Open(filePath)
			if err != nil {
				global.Fatalf("failed to read file %v: %s", filePath, err)
			}
			sw.write(f)
			sw.writeString("`\n")
		}
	}
	sw.writeString(")\n")

	tmpName := tmpF.Name()
	_ = tmpF.Close()

	// We can't use os.Rename here because it gives "The system cannot move the file to a different disk drive"
	// error on Windows. Temp directory can be on drive C:\ and the program can be executed on D:\
	err = copyFile(tmpName, outFileName)
	if err != nil {
		global.Fatalf("failed to copy file from %v to %v: %s", tmpF.Name(), outFileName, err)
	}

	cwd, _ := os.Getwd()
	_, _ = fmt.Fprintf(os.Stderr, "Generated file: %v (%v)\n", outFileName, cwd)
}

type strictWriter struct {
	w io.Writer
}

func (sw *strictWriter) writeString(s string) {
	_, err := sw.w.Write([]byte(s))
	if err != nil {
		panic(err)
	}
}

func (sw *strictWriter) write(r io.Reader) {
	_, err := io.Copy(sw.w, r)
	if err != nil {
		panic(err)
	}
}

var preambula = `
/*
DO NOT EDIT!
This code was generated automatically using _swagger-helper
*/

`
