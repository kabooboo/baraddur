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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/sirupsen/logrus" // TODO: use ZAP instead
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Reset = "\033[0m"
	Red   = "\033[31m"
	Green = "\033[32m"
)

var (
	logger      = logrus.New()
	configFile  string
	outputColor string
	scanCmd     = &cobra.Command{
		Use:   "scan <target-directory>",
		Short: "Scan a directory recursively and execute code",
		Long: `Recursively iterate over a directory and execute an arbitrary
command whenever a regexp match is found, based on the configuration file.`,
		Args: cobra.ExactArgs(1),
		Run:  runCommand,
	}
)

func init() {

	// Cobra configuration
	scanCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to config file. Defaults to ~/.baraddur/config.yaml")
	scanCmd.PersistentFlags().StringP("log-level", "l", "info", "LogLevel for the CLI. One of \"error\", \"info\", \"debug\" or \"trace\".")
	scanCmd.PersistentFlags().StringP("output", "o", "colored", "How to print the command's outouts. One of \"colored\", \"no-color\" or \"none\".")
	scanCmd.PersistentFlags().StringP("job", "j", "", "Specify a job to run.")
	scanCmd.PersistentFlags().BoolP("dry-run", "d", false, "Scan without executing commands")
	scanCmd.PersistentFlags().IntP("workers", "w", 5, "Number of parallel workers to use for command execution")

	rootCmd.AddCommand(scanCmd)

	// Viper flag configuration
	viper.BindPFlag("log-level", scanCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output", scanCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("job", scanCmd.PersistentFlags().Lookup("job"))
	viper.BindPFlag("dry-run", scanCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("workers", scanCmd.PersistentFlags().Lookup("workers"))

}

func initConfig() {

	// Viper configuration
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.baraddur/config.yaml"
	}

	viper.SetConfigFile(configFile)

	envReplacer := strings.NewReplacer("-", "_")

	viper.SetEnvPrefix(envReplacer.Replace(programName))
	viper.SetEnvKeyReplacer(envReplacer)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		logger.Info("Using config file: ", viper.ConfigFileUsed())
	} else {
		logger.Error("Couldn't load config: ", err)
	}

	// Logrus configuration
	logger.SetFormatter(&logrus.TextFormatter{})
	logger.SetOutput(os.Stdout)

	switch viper.GetString("log-level") {
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "trace":
		logger.SetLevel(logrus.TraceLevel)
	default:
		logger.Error("Invalid log-level argument: \"", viper.GetString("log-level"), "\". Defaulting to \"info\".")
		logger.SetLevel(logrus.InfoLevel)
	}

	switch viper.GetString("output") {
	case "colored":
		outputColor = "colored"
	case "no-color":
		outputColor = "no-color"
	case "none":
		outputColor = "none"
	default:
		logger.Error("Invalid output argument: \"", viper.GetString("output"), "\". Defaulting to \"none\".")
		outputColor = "none"
	}
}

type CommandJob struct {
	Command     string
	Args        []string
	TriggerPath string
}

type ScanJob struct {
	Name    string
	Pattern string
	Command []string
}

type ScanConfig struct {
	Jobs []ScanJob
}

func runCommand(cmd *cobra.Command, args []string) {

	initConfig()

	root := args[0]
	scanConfig := ScanConfig{}
	viper.Unmarshal(&scanConfig)
	logrus.WithFields(logrus.Fields{"root": root}).Info("Starting Baraddur")

	var workerWaitGroup sync.WaitGroup
	var scannerWaitGroup sync.WaitGroup

	jobs := make(chan *CommandJob)

	for _, job := range scanConfig.Jobs {

		pattern_str := job.Pattern
		command := job.Command
		name := job.Name

		if viper.GetString("job") != "" && viper.GetString("job") != name {
			continue
		}

		logrus.WithFields(logrus.Fields{"root": root, "job": name, "pattern": pattern_str}).Info("Starting job")

		re, err := regexp.Compile(pattern_str)

		if err != nil {
			logrus.WithFields(
				logrus.Fields{"regexp": pattern_str, "error": err.Error()},
			).Error("Couldn't parse regexp")
			// return?
		}

		logrus.WithFields(logrus.Fields{"regexp": pattern_str}).Debug("Found Regexp")

		logrus.Debug("Starting WaitGroups")

		logrus.WithFields(logrus.Fields{"workers": viper.GetInt("workers")}).Debug("Starting workers")
		for w := 1; w <= viper.GetInt("workers"); w++ {
			workerWaitGroup.Add(1)
			go worker(w, jobs, &workerWaitGroup)
		}

		// TODO: Make scan parallelizable
		scannerWaitGroup.Add(1)
		walkDir(root, re, &command, jobs, &scannerWaitGroup)

		logrus.WithFields(logrus.Fields{"root": root, "job": name, "pattern": pattern_str}).Info("Job done")

	}

	scannerWaitGroup.Wait()
	close(jobs)
	workerWaitGroup.Wait()

	logrus.WithFields(logrus.Fields{"root": root}).Info("Baraddur done")

}

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:strings.LastIndex(s[:max], " ")]
}

func walkDir(dir string, re *regexp.Regexp, commandAndArgsTemplate *[]string, commandJobs chan *CommandJob, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO: find a means to differentiate files from directories.
	// Perhaps prepend all paths with some sort of prefix? `file:path/to/file` ?
	visit := func(path string, f os.FileInfo, err error) error {

		matches := re.FindStringSubmatch(path)

		if matches != nil {

			job := new(CommandJob)

			job.TriggerPath = path

			command_and_args := make([]string, len(*commandAndArgsTemplate))
			for i, v := range *commandAndArgsTemplate {
				logger.Debug(v)
				command_and_args[i] = re.ReplaceAllString(path, v)
			}
			job.Command = (command_and_args)[0]
			job.Args = (command_and_args)[1:]

			logrus.WithFields(logrus.Fields{"match": path}).Debug("Found match")
			commandJobs <- job
		}

		if f.IsDir() && path != dir {
			wg.Add(1)

			go walkDir(path, re, commandAndArgsTemplate, commandJobs, wg)
			return filepath.SkipDir
		}
		if f.Mode().IsRegular() {

		}
		return nil
	}

	filepath.Walk(dir, visit)
}

func worker(id int, jobs <-chan *CommandJob, wg *sync.WaitGroup) {

	defer wg.Done()

	for j := range jobs {
		logger.WithFields(logrus.Fields{
			"worker":   id,
			"job_path": j.TriggerPath,
			"command":  j.Command,
			"args":     j.Args,
		}).Trace("Command will run")

		cmd := exec.Command(j.Command, j.Args...)

		coloredOutputs := []string{"colored", "no-color"}

		if slices.Contains(coloredOutputs, outputColor) {
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()

			_ = cmd.Start()
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					if outputColor == "colored" {
						fmt.Fprintf(os.Stdout, Green)
					}
					fmt.Fprintln(os.Stdout, scanner.Text())
					if outputColor == "colored" {
						fmt.Fprintf(os.Stdout, Reset)
					}
				}

			}()
			go func() {
				defer wg.Done()
				scanner := bufio.NewScanner(stderr)

				for scanner.Scan() {
					if outputColor == "colored" {
						fmt.Fprintf(os.Stderr, Red)
					}
					fmt.Fprintln(os.Stderr, scanner.Text())
					if outputColor == "colored" {
						fmt.Fprintf(os.Stdout, Reset)
					}
				}
			}()
			wg.Wait()
		} else {
			_ = cmd.Start()
		}

		err := cmd.Wait()

		if err != nil {
			logger.WithFields(logrus.Fields{"worker": id, "job_path": j.TriggerPath, "output": "stderr", "goerror": err}).Error("Command exited with errors")
		}

		logger.WithFields(logrus.Fields{"worker": id, "job_path": j.TriggerPath}).Debug("Command exited succcessfully")
	}
}
