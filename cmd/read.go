// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
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

package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Generate a pete file based on a protobuf input file",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("read called")
		fmt.Println("READ DOES NOT WORK YET")
		workingDir, _ := os.Getwd()
		input := path.Join(workingDir, viper.GetString("input"))
		output := path.Join(workingDir, viper.GetString("output"))
		deli := strings.Replace(viper.GetString("deli"), "\\n", "\n", -1)
		all := viper.GetBool("all")
		if !all {
		}
		names := viper.GetStringSlice("names")
		_ = input
		_ = output
		_ = deli
		_ = names

	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	readCmd.Flags().StringP("input", "i", "", ".proto file to read queries from")
	readCmd.Flags().StringP("output", "o", "", ".pete file to write queries to")
	readCmd.Flags().StringSliceP("names", "n", nil, "names of the queries to read")
	readCmd.Flags().BoolP("all", "a", false, "read all the queries")
}
