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
	"io/ioutil"
	"strings"

	absp "github.com/rhysd/abspath"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// readCmd represents the read command
var readCmd = &cobra.Command{
	Use:   "read",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("read called")
		input, err := absp.ExpandFrom(viper.GetString("read-input"))
		if err != nil {
			panic(fmt.Sprintf("error expanding input: %+v", err))
		}
		output, err := absp.ExpandFrom(viper.GetString("read-output"))
		if err != nil {
			panic(fmt.Sprintf("error expanding output: %+v, %+v", err, viper.GetString("read-output")))
		}
		deli := strings.Replace(viper.GetString("deli"), "\\n", "\n", -1)
		names := viper.GetStringSlice("read-names")
		// prefix to cut
		// TODO: make []string?
		prefix := viper.GetString("prefix")
		all := viper.GetBool("read-all")

		fmt.Println("input: ", input)
		fmt.Println("output: ", output)
		fmt.Println("deli: ", deli)
		fmt.Println("prefix: ", prefix)
		fmt.Println("all: ", all)

		rawProtoQueries, err := protoQueriesFromFile(input.String())
		if err != nil {
			panic(err)
		}
		rawPeteQueries, err := peteQueriesFromFile(output.String(), deli)
		if err != nil {
			panic(err)
		}
		peteQueries := make([]*querySerializer, len(rawPeteQueries))
		for i, v := range rawPeteQueries {
			peteQueries[i] = newQuerySerializerFromPete(v, "", prefix)
		}
		protoQueries := make([]*querySerializer, len(rawProtoQueries))
		for i, v := range rawProtoQueries {
			protoQueries[i] = newQuerySerializerFromProto(v, "", prefix)
		}
		// if our names are equal, or prefix + n == our name, or all is true
		// we want this query to replace pete's
		keepQuery := func(name string) bool {
			for _, n := range names {
				if n == strings.TrimPrefix(name, prefix) || n == name || all {
					return true
				}
			}
			return false
		}
		// replaces peteQuery with same name, or appends it
		replaceOrAppend := func(protoQ *querySerializer) {
			for i, peteQ := range peteQueries {
				if peteQ.name == protoQ.name {
					peteQueries[i] = protoQ
					return
				}
			}
			peteQueries = append(peteQueries, protoQ)
		}

		// loop through all the protoQueries, if they match any of our names
		// or the all option is true, replace our pete query of the same name
		for _, protoQ := range protoQueries {
			if keepQuery(protoQ.name) {
				replaceOrAppend(protoQ)
			}
		}
		// write out all the peteQueries now, they should be in the right order
		toJoinSlice := make([]string, len(peteQueries))
		for i, v := range peteQueries {
			toJoinSlice[i] = v.ToPete()
		}

		data := strings.Join(toJoinSlice, deli)
		if err = ioutil.WriteFile(output.String(), []byte(data), 0644); err != nil {
			panic(err)
		}

	},
}

func init() {
	rootCmd.AddCommand(readCmd)

	readCmd.Flags().StringP("input", "i", "", ".proto file to read queries from")
	readCmd.Flags().StringP("output", "o", "", ".pete file to write queries to")
	//rootCmd.Flags().StringP("deli", "d", "\n\n", "the delimiter to use")
	readCmd.Flags().StringSliceP("names", "n", nil, "names of the queries to read")
	readCmd.Flags().BoolP("all", "a", false, "read all the queries")

	// prefixed because of https://github.com/spf13/viper/issues/567
	viper.BindPFlag("read-input", readCmd.Flags().Lookup("input"))
	viper.BindPFlag("read-output", readCmd.Flags().Lookup("output"))
	viper.BindPFlag("read-names", readCmd.Flags().Lookup("names"))
	viper.BindPFlag("read-prefix", readCmd.Flags().Lookup("prefix"))
	viper.BindPFlag("read-all", readCmd.Flags().Lookup("all"))
	//viper.BindPFlag("deli", rootCmd.PersistentFlags().Lookup("deli"))
}
