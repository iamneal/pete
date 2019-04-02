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
	Short: "Generate a pete file based on a protobuf input file",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
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
		peteQueries := make([]*querySerializer, 0)
		for _, v := range rawPeteQueries {
			peteQueries = append(peteQueries, newQuerySerializerFromPete(v, "", prefix))
		}
		protoQueries := make([]*querySerializer, 0)
		for _, v := range rawProtoQueries {
			protoQueries = append(protoQueries, newQuerySerializerFromProto(v, "", prefix))
		}
		// if our names are equal, or all is true
		// we want this query to replace pete's
		keepQuery := func(name string) bool {
			if all {
				return true
			}
			for _, n := range names {
				if n == strings.TrimSpace(name) || n == name {
					return true
				}
			}
			return false
		}
		// replaces peteQuery with same name, or appends it
		replaceOrAppend := func(protoQ *querySerializer) {
			for i, peteQ := range peteQueries {
				fmt.Printf("names equal?\npete:  %v\nproto: %v\n", peteQ.name, protoQ.name)
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
		data = strings.TrimSpace(data)
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
