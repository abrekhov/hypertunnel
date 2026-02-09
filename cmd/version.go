/*
Copyright © 2021 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These variables are intended to be set at build time via -ldflags.
// Defaults are useful for local `go build`.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		long, _ := cmd.Flags().GetBool("long")
		if !long {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "version\t%s\ncommit\t%s\ndate\t%s\n", Version, Commit, Date)
	},
}

func init() {
	// Keep `--version` output copy/paste friendly (one line).
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	versionCmd.Flags().Bool("long", false, "Print extended version information")
	rootCmd.AddCommand(versionCmd)
}
