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
	"unicode"

	absp "github.com/rhysd/abspath"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// writeCmd will provide all the functionality needed to write to proto queries
var writeCmd = &cobra.Command{
	Use:   "write <input> <output>",
	Short: "Used to write to the persist query options",
	Long: `The write command will read from an input pete
file and write the formatted output to the proto queries option`,

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Usage()
			return
		}

		input, err := absp.ExpandFrom(args[0])
		if err != nil {
			fmt.Printf("error expanding input: %+v", err)
			return
		}

		output, err := absp.ExpandFrom(args[1])
		if err != nil {
			fmt.Printf("error expanding output: %+v", err)
			return
		}

		deli := strings.Replace(viper.GetString("write-deli"), "\\n", "\n", -1)
		linepad := viper.GetString("linepad")
		prefix := viper.GetString("write-prefix")
		tabsize := "  "

		fmt.Println("input: ", input)
		fmt.Println("output: ", output)
		fmt.Println("deli: ", deli)
		fmt.Print("prefix: ", prefix, "\n\n")

		protofile, queryStart, queryEnd, err := protoFileQueriesPos(output.String())
		if err != nil {
			fmt.Println(err)
			return
		}

		// get unformatted queries from pete file
		queries, err := peteQueriesFromFile(input.String(), deli)
		if err != nil {
			fmt.Println(err)
			return
		}

		// now format our queries
		decoratePeteQueries(queries, linepad, prefix, tabsize)

		joinedQueries := strings.Join(queries, "")

		data := protofile[0:queryStart] +
			header(linepad) +
			joinedQueries +
			footer(linepad) +
			protofile[queryEnd:]

		if err = ioutil.WriteFile(output.String(), []byte(data), 0644); err != nil {
			fmt.Println(err)
			return
		}
	},
}

type querySerializer struct {
	padding string
	prefix  string
	name    string
	inline  string
	outline string
	query   []string
}

func newQuerySerializer(queryParts []string, padding, prefix string) *querySerializer {
	q := new(querySerializer)
	// first line is always the name
	q.name = queryParts[0]
	queryParts = queryParts[1:]

	q.padding = padding
	q.prefix = prefix
	// the line that contains "in: " is the input
	// the line taht contains "out: " is the output
	for _, v := range queryParts {
		if strings.HasPrefix(v, "in: ") {
			q.inline = strings.TrimPrefix(v, "in: ")
		} else if strings.Contains(v, "out: ") {
			q.outline = strings.TrimPrefix(v, "out: ")
		} else {
			q.query = append(q.query, v)
		}
	}
	return q
}

/*
	padding + { + \n
	padding + \t + name: + " + VALUE + ", + \n
	padding + \t + query: [ + \n + VALUE + ], + \n
	padding + \t + " + prefix + VALUE + ", + \n
	padding + \t + pm_strategy: + " + $ + "
	padding + } + ,
*/
func (q *querySerializer) Serialize(tabsize string) string {
	var decoratedQuery, squashed string
	comma := func(i int) string {
		if i == len(q.query)-1 {
			return ""
		}
		return ","
	}
	typename := func(s string) string {
		if strings.Contains(s, ".") || q.prefix == "" { // we must already have a full annotated path, so no adjustment
			return s
		}
		return q.prefix + "." + s
	}

	queryStringTab := tabsize + tabsize
	insideBraceTab := tabsize + tabsize + tabsize
	for i, v := range q.query {
		// we need to make this line pretty
		spaces, line := trimLeftAndKeepSpaces(v)
		squashed += q.padding + queryStringTab + spaces + fmt.Sprintf(`"%s"`, line) + comma(i) + "\n"
	}

	// TODO make }, { optional
	decoratedQuery += q.padding + "{\n"
	// TODO make the tab size adjustable
	decoratedQuery += insideBraceTab + fmt.Sprintf(`name: "%s",`+"\n", q.name)
	decoratedQuery += insideBraceTab + "query: [\n"
	decoratedQuery += squashed
	decoratedQuery += insideBraceTab + "],\n"
	// TODO make this optional too
	decoratedQuery += insideBraceTab + `pm_strategy: "$",` + "\n"
	decoratedQuery += insideBraceTab + fmt.Sprintf(`in: "%s",`+"\n", typename(q.inline))
	decoratedQuery += insideBraceTab + fmt.Sprintf(`out: "%s",`+"\n", typename(q.outline))
	decoratedQuery += q.padding + "}"

	return decoratedQuery
}

func header(padding string) string {
	return padding + "queries: [\n"
}

func footer(padding string) string {
	return padding + "];\n"
}

func trimLeftAndKeepSpaces(s string) (spaces string, trimmed string) {
	trimmed = strings.TrimLeftFunc(s, func(r rune) bool {
		if unicode.IsSpace(r) {
			spaces += string(r)
			return true
		}
		return false
	})
	return
}

func init() {
	rootCmd.AddCommand(writeCmd)

	/* Flags
	 * deli    - the delimeter to seperate pete queries
	 * linepad - the padding to be used in the proto file
	 * prefix  - the package prefix for in and out types
	 */
}
