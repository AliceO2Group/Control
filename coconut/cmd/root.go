/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package cmd

import (
	"os"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"path"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "coconut",
	Short: "O² Control and Configuration Utility",
	Long: `The O² Control and Configuration utility is a command line program for interacting with the O² Control System.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.WithField("error", err).Fatal("cannot run command")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	viper.Set("version", "0.1")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file (default $HOME/.config/coconut/settings.yaml)")
	rootCmd.PersistentFlags().String("endpoint", "127.0.0.1:47102", "O² Control endpoint as HOST:PORT")
	rootCmd.PersistentFlags().String("config_endpoint", "consul://127.0.0.1:8500", "O² Configuration endpoint as PROTO://HOST:PORT")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "show verbose output for debug purposes")

	viper.BindPFlag("endpoint", rootCmd.PersistentFlags().Lookup("endpoint"))
	viper.BindPFlag("config_endpoint", rootCmd.PersistentFlags().Lookup("config_endpoint"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetDefault("log.level", "info")
	viper.SetDefault("endpoint", "127.0.0.1:47102")
	viper.SetDefault("config_endpoint", "127.0.0.1:8500")
	viper.SetDefault("verbose", false)

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.WithField("error", err).Error("cannot find configuration file")
			os.Exit(1)
		}

		// Search config in .config/coconut directory with name "settings.yaml"
		viper.AddConfigPath(path.Join(home, ".config/coconut"))
		viper.SetConfigName("settings")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logLevel, err := logrus.ParseLevel(viper.GetString("log.level"))
		if err == nil {
			logrus.SetLevel(logLevel)
		}
		log.WithField("file", viper.ConfigFileUsed()).
			Debug("configuration loaded")
	}

	if viper.GetBool("verbose") {
		viper.Set("log.level", "debug")
		logrus.SetLevel(logrus.DebugLevel)
	}
}
