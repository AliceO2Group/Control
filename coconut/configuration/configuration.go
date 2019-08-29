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

// Package configuration handles the details of interfacing with
// the O² Configuration store.
package configuration

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/briandowns/spinner"
	"time"
	"io"
	"github.com/sirupsen/logrus"
	"os"
	"github.com/AliceO2Group/Control/common/logger"
	"fmt"
	"strings"
	"github.com/AliceO2Group/Control/configuration"
	"errors"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"github.com/naoina/toml"
)

var log = logger.New(logrus.StandardLogger(), "coconut")

type RunFunc func(*cobra.Command, []string)

type ConfigurationCall func(configuration.Source, *cobra.Command, []string, io.Writer) (error)

var componentsPath = "o2/components/"

func WrapCall(call ConfigurationCall) RunFunc {
	return func(cmd *cobra.Command, args []string) {
		endpoint := viper.GetString("config_endpoint")
		log.WithPrefix(cmd.Use).
			WithField("config_endpoint", endpoint).
			Debug("initializing configuration client")

		s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
		s.Color("yellow")
		s.Suffix = " working..."
		s.Start()

		cfg, err := configuration.NewSource(endpoint)
		if err != nil {
			var fields logrus.Fields
			if logrus.GetLevel() == logrus.DebugLevel {
				fields = logrus.Fields{"error": err}
			}
			log.WithPrefix(cmd.Use).
				WithFields(fields).
				Fatal("cannot query endpoint")
			os.Exit(1)
		}

		var out strings.Builder

		// redirect stdout to null, the only way to output is
		stdout := os.Stdout
		os.Stdout,_ = os.Open(os.DevNull)
		err = call(cfg, cmd, args, &out)
		os.Stdout = stdout
		s.Stop()
		if err != nil {
			var fields logrus.Fields
			if logrus.GetLevel() == logrus.DebugLevel {
				fields = logrus.Fields{"error": err}
			}
			log.WithPrefix(cmd.Use).
				WithFields(fields).
				Fatal("cannot query endpoint")
			os.Exit(1)
		}

		fmt.Print(out.String())
	}
}

func Dump(cfg configuration.Source, cmd *cobra.Command, args []string, o io.Writer) (err error) {
	if len(args) != 1 {
		err = errors.New(fmt.Sprintf("accepts 1 arg(s), received %d", len(args)))
		return
	}
	key := args[0]

	data, err := cfg.GetRecursive(key)
	if err != nil {
		return
	}

	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return
	}

	var output []byte
	switch strings.ToLower(format) {
	case "json":
		output, err = json.MarshalIndent(data, "", "    ")
	case "yaml":
		output, err = yaml.Marshal(data)
	case "toml":
		output, err = toml.Marshal(data)
	}
	if err != nil {
		log.WithField("error", err.Error()).Fatalf("cannot serialize subtree to %s", strings.ToLower(format))
		return
	}

	fmt.Fprintln(o, string(output))

	return nil
}

func List(cfg configuration.Source, cmd *cobra.Command, args []string, o io.Writer)(err error) {
	if len(args) > 1 {
		err = errors.New(fmt.Sprintf("command requires maximum 1 arg but received %d", len(args)))
		return
	}

	keyPrefix := componentsPath
	if len(args) > 0 {
		keyPrefix += args[0] + "/"
	}

	useTimestamp, err := cmd.Flags().GetBool("timestamp")

	keys, err := cfg.GetKeysByPrefix(keyPrefix, "")
	if err != nil {
		return
	}
	var components []string

	componentsSet := make(map[string]string)
	for _, key := range keys {
		componentsFullName := strings.TrimPrefix(key, keyPrefix)
		componentParts := strings.Split(componentsFullName, "/")
		componentTimestamp := componentParts[len(componentParts) - 1]
		if useTimestamp {
			componentsFullName = strings.TrimSuffix(componentsFullName, "/"+componentTimestamp)
		} else {
			componentsFullName = componentParts[0]
		}

		if strings.Compare(componentsSet[componentsFullName], componentTimestamp) < 0{
			componentsSet[componentsFullName] = componentTimestamp
		}
	}

	for key,value := range componentsSet {
		if useTimestamp {
			components = append(components, key+"/"+value)
		} else {
			components = append(components, key)
		}
	}

	output, err := formatOutput(cmd, components)
	if err != nil {
		return
	}
	fmt.Fprintln(o, string(output))
	return nil
}

func formatOutput( cmd *cobra.Command, output []string)(parsedOutput []byte, err error) {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return
	}

	switch strings.ToLower(format) {
	case "json":
		parsedOutput, err = json.MarshalIndent(output, "", "    ")
	case "yaml":
		parsedOutput, err = yaml.Marshal(output)
	case "toml":
		parsedOutput, err = toml.Marshal(output)
	}
	if err != nil {
		log.WithField("error", err.Error()).Fatalf("cannot serialize subtree to %s", strings.ToLower(format))
		return
	}
	return parsedOutput, nil
}