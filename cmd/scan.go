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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	logger     = logrus.New()
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

	// Cobra configuration
	scanCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "./baraddur.yml", "Path to config file.")
	scanCmd.PersistentFlags().BoolP("debug", "", false, "Print debugging information")
	scanCmd.PersistentFlags().BoolP("dry-run", "", false, "Scan without executing commands")
	scanCmd.PersistentFlags().IntP("workers", "", 5, "Number of parallel workers to use")

	rootCmd.AddCommand(scanCmd)

	// Viper flag configuration
	viper.BindPFlag("debug", scanCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("dry-run", scanCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("workers", scanCmd.PersistentFlags().Lookup("workers"))

}

func initConfig() {

	// Viper configuration
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
	logger.SetFormatter(&log.TextFormatter{})
	logger.SetOutput(os.Stdout)

	if viper.GetBool("debug") {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}

}

type Job struct {
	Command     string
	Args        []string
	TriggerPath string
}

func runCommand(cmd *cobra.Command, args []string) {

	initConfig()

	root := "/home/gcreti/Projects"      // example - WIP
	exp_str := `^(.*requirements\.txt)$` // example - WIP
	command_template := []string{`bash`, `-c`, `cat $1`}

	re, err := regexp.Compile(exp_str)

	if err != nil {
		log.WithFields(
			log.Fields{"regexp": exp_str, "error": err.Error()},
		).Error("Couldn't parse regexp")
		// return?
	}

	log.WithFields(log.Fields{"regexp": exp_str}).Info("Found Regexp")

	jobs := make(chan *Job)

	log.Debug("Starting WaitGroups")
	var workerWaitGroup sync.WaitGroup
	var scannerWaitGroup sync.WaitGroup

	log.WithFields(log.Fields{"workers": viper.GetInt("workers")}).Info("Starting workers")
	for w := 1; w <= viper.GetInt("workers"); w++ {
		workerWaitGroup.Add(1)
		go worker(w, jobs, &workerWaitGroup)
	}

	scannerWaitGroup.Add(1)
	walkDir(root, re, &command_template, jobs, &scannerWaitGroup)

	workerWaitGroup.Wait()
	scannerWaitGroup.Wait()
	close(jobs)
}

func walkDir(dir string, re *regexp.Regexp, command_and_args_template *[]string, jobs chan *Job, wg *sync.WaitGroup) {
	defer wg.Done()

	visit := func(path string, f os.FileInfo, err error) error {

		matches := re.FindStringSubmatch(path)

		if matches != nil {

			job := new(Job)

			job.TriggerPath = path

			command_and_args := make([]string, len(*command_and_args_template))
			for i, v := range *command_and_args_template {
				logger.Debug(v)
				command_and_args[i] = re.ReplaceAllString(path, v)
			}
			job.Command = (command_and_args)[0]
			job.Args = (command_and_args)[1:]

			log.WithFields(log.Fields{"match": path}).Debug("Found match")
			jobs <- job
		}

		if f.IsDir() && path != dir {
			wg.Add(1)

			go walkDir(path, re, command_and_args_template, jobs, wg)
			return filepath.SkipDir
		}
		if f.Mode().IsRegular() {

		}
		return nil
	}

	filepath.Walk(dir, visit)
}

func worker(id int, jobs <-chan *Job, wg *sync.WaitGroup) {

	defer wg.Done()

	for j := range jobs {
		logger.WithFields(log.Fields{"worker": id, "job_path": j.TriggerPath}).Info("Started handling match")
		logger.WithFields(log.Fields{
			"worker":   id,
			"job_path": j.TriggerPath,
			"command":  j.Command,
			"args":     j.Args,
		}).Info("Will run command")

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd := exec.Command(j.Command, j.Args...)

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()

		if err != nil {
			logger.WithFields(
				log.Fields{
					"worker":   id,
					"job_path": j.TriggerPath,
					"stderr":   stderr.String(),
					"out":      stdout.String(),
					"goerror":  err,
				}).Error("Could not run command")
		}
		logger.WithFields(log.Fields{"worker": id, "job_path": j.TriggerPath}).Debug("Output: ", stdout.String())

		logger.WithFields(log.Fields{"worker": id, "job_path": j.TriggerPath}).Info("Finished handling match")
	}
}
