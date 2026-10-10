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

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	var long bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if long {
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "version\t%s\ncommit\t%s\ndate\t%s\n", version, commit, date)
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
	cmd.Flags().BoolVar(&long, "long", false, "Print extended version information")
	rootCmd.AddCommand(cmd)
}
