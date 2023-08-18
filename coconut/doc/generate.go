/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package main

import (
	"os"

	"github.com/AliceO2Group/Control/coconut/app"
	"github.com/AliceO2Group/Control/coconut/cmd"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra/doc"
)

var log = logger.New(logrus.StandardLogger(), app.NAME)

func main() {
	rootCmd := cmd.GetRootCmd()

	// Generate Markdown docs
	err := doc.GenMarkdownTree(rootCmd, "./")
	if err != nil {
		log.Fatal(err)
	}

	// Generate man pages
	header := &doc.GenManHeader{
		Title:   "ALIECS",
		Section: "1",
	}

	const manPath = "../../local/share/man"

	_ = os.MkdirAll(manPath, 0755)
	err = doc.GenManTree(rootCmd, header, manPath)
	if err != nil {
		log.Fatal(err)
	}

	// Generate bash completion file
	const bashCompletionPath = "../../share/bash-completion/completions"
	_ = os.MkdirAll(bashCompletionPath, 0755)
	err = rootCmd.GenBashCompletionFileV2(bashCompletionPath+"/coconut.sh", true)
	if err != nil {
		log.Fatal(err)
	}

	const zshCompletionPath = "../../share/zsh/site-functions"
	_ = os.MkdirAll(zshCompletionPath, 0755)
	err = rootCmd.GenZshCompletionFile(bashCompletionPath + "/_coconut")
	if err != nil {
		log.Fatal(err)
	}

}
