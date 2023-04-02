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
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	scanCmd    = &cobra.Command{
		Use:   "scan <target-directory>",
		Short: "Scan a directory recursively and execute code",
		Long: `Recursively iterate over a directory and execute an arbitrary
command whenever a regexp match is found, based on the configuration file.`,
		Args: cobra.ExactArgs(1),
		Run:  runCommand,
	}
)

func init() {

	scanCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./baraddur.yml", "Path to config file.")
	scanCmd.PersistentFlags().BoolP("debug", "", false, "Print debugging information")
	scanCmd.PersistentFlags().BoolP("dry-run", "", false, "Scan without executing commands")

	scanCmd.MarkPersistentFlagRequired("config")

	rootCmd.AddCommand(scanCmd)

}

func initConfig() {
	viper.SetConfigFile(configFile)

	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		log.Println("Unable to bind pflags:", err)
	}

	envReplacer := strings.NewReplacer("-", "_")

	viper.SetEnvPrefix(envReplacer.Replace(programName))
	viper.SetEnvKeyReplacer(envReplacer)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}

	log.SetFlags(0)

	// TODO: figure logrus out

}

func runCommand(cmd *cobra.Command, args []string) {
	fmt.Println(fmt.Println(args))

	initConfig()

	// TODO: implementation
}
