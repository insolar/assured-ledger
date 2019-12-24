//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package version

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version     = "unset" // Version is release semantic version.
	BuildNumber = "unset" // BuildNumber is CI build number.
	BuildDate   = "unset" // BuildDate is build date.
	BuildTime   = "unset" // BuildTime is build date.
	CITool      = "unset" // CITool is a continuous integration tool(Travis, DockerCloud, etc.).
	GitHash     = "unset" // GitHash is short git commit hash.
)

// GetFullVersion returns multi line full version information
func GetFullVersion() string {

	result := fmt.Sprintf(`
 Version      : %s
 Build number : %s
 Build date   : %s %s
 Git hash     : %s
 Go version   : %s
 Go compiler  : %s
 Platform     : %s/%s`, Version, BuildNumber, BuildDate, BuildTime, GitHash, runtime.Version(),
		runtime.Compiler, runtime.GOOS, runtime.GOARCH)

	return result
}

func GetCommand(cmdName string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print the version info of %s", cmdName),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(GetFullVersion())
			os.Exit(0)
		},
	}
}
