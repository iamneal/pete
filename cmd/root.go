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

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pete",
	Short: "A cli tool for persist query option generation/maintenance",
	Long:  `This is a cli tool that will enable you to generate the options for protoc-gen-persist queries`,

	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
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

	// Load the config file flag directly into the cfgFile var
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (if no file is provided, uses $HOME/.pete)")

	/* Flags
	 * Any flags defined in the root command will be
	 * shared across both the read and the write commands
	 *
	 * deli    - the delimeter to seperate pete queries
	 * linepad - the padding to be used in the proto file
	 * prefix  - the package prefix for in and out types
	 */

	rootCmd.PersistentFlags().StringP("deli", "d", "\n\n", "the delimiter to use")
	rootCmd.PersistentFlags().StringP("linepad", "l", "    ", "the padding string for each line")
	rootCmd.PersistentFlags().StringP("prefix", "p", "", "the package prefix for your in and out types")
	viper.BindPFlag("deli", rootCmd.PersistentFlags().Lookup("deli"))
	viper.BindPFlag("linepad", rootCmd.PersistentFlags().Lookup("linepad"))
	viper.BindPFlag("prefix", rootCmd.PersistentFlags().Lookup("prefix"))
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
