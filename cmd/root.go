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
	"os"
	"path"
	"strings"
	"unicode"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pete",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		workingDir, _ := os.Getwd()
		input := path.Join(workingDir, viper.GetString("input"))
		output := path.Join(workingDir, viper.GetString("output"))
		deli := strings.Replace(viper.GetString("deli"), "\\n", "\n", -1)
		linepad := viper.GetString("linepad")
		prefix := viper.GetString("prefix")
		tabsize := "  "

		fmt.Println("input: ", input)
		fmt.Println("output: ", output)
		fmt.Println("deli: ", deli)
		fmt.Println("prefix: ", prefix)

		file, err := ioutil.ReadFile(input)
		if err != nil {
			panic(err)
		}
		pBytes, err := ioutil.ReadFile(output)
		if err != nil {
			panic(err)
		}
		persistFile := string(pBytes)

		lineWithPersist := strings.Index(persistFile, "persist.ql")
		if lineWithPersist < 0 {
			panic(fmt.Errorf("not a persist file"))
		}
		// find next nearest newline, that is where we will start our search
		nextNl := strings.Index(persistFile[lineWithPersist:], "\n")
		if nextNl < 0 {
			panic(fmt.Errorf("not a finished persist file"))
		}

		// represents the number of '{' on the stack.  Every '{' rune incs, and '}' decs
		// goal being to get this to zero before eof, that is the queries we need to replace
		i := 0
		for braceStack := 1; braceStack != 0; i++ {
			c := persistFile[lineWithPersist+nextNl+i]
			if c == '{' {
				braceStack++
			} else if c == '}' {
				braceStack--
			}
		}
		// line with opts, + offset of newline, + 1 to include the newline
		queryStart := lineWithPersist + nextNl + 1
		// line with opts, + offset of newline, + till closing curly brace -1
		queryEnd := lineWithPersist + nextNl + i - 1

		// now format our queries from the file we read
		queriesOpts := strings.Split(string(file), deli)
		var queries []string
		for i, v := range queriesOpts {
			if strings.TrimSpace(v) == "" {
				continue
			}
			queryParts := strings.Split(v, "\n")
			// TODO filter out bad lines
			serilizer := newQuerySerializer(queryParts, linepad, prefix)
			decoratedQuery := serilizer.Serialize(tabsize)
			// this isn't the last query so add a comma
			if i < len(queriesOpts)-1 {
				decoratedQuery += ","
			}
			decoratedQuery += "\n"

			queries = append(queries, decoratedQuery)
		}
		joinedQueries := strings.Join(queries, "")

		data := persistFile[0:queryStart] +
			header(linepad) +
			joinedQueries +
			footer(linepad) +
			persistFile[queryEnd:]

		if err = ioutil.WriteFile(output, []byte(data), 0644); err != nil {
			panic(err)
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pete.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("input", "i", "persist.pete", "file to parse (default is \"persist.pete\"")
	rootCmd.Flags().StringP("output", "o", "", "file to write to")
	rootCmd.Flags().StringP("deli", "d", "\n\n", "the delimiter to use")
	rootCmd.Flags().StringP("linepad", "l", "    ", "the padding string for each line defaults to 4 spaces")
	rootCmd.Flags().StringP("prefix", "p", "", "the package prefix for your in and out types")
	viper.BindPFlag("input", rootCmd.Flags().Lookup("input"))
	viper.BindPFlag("output", rootCmd.Flags().Lookup("output"))
	viper.BindPFlag("deli", rootCmd.Flags().Lookup("deli"))
	viper.BindPFlag("linepad", rootCmd.Flags().Lookup("linepad"))
	viper.BindPFlag("prefix", rootCmd.Flags().Lookup("prefix"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		fmt.Println("parsing config flag: ", cfgFile)
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".pete" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".pete")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
