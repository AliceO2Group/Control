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

package product

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func getExecutableDir() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	exPath := filepath.Dir(ex)
	return exPath
}

func fillVersionFromVersionFile(versionFilePath string) {
	vfContents, err := ioutil.ReadFile(versionFilePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(vfContents), "\n")

	for _, line := range lines {
		if !strings.HasPrefix(line, "VERSION_") {
			continue
		}
		splitLine := strings.Split(line, ":=")
		if len(splitLine) != 2 {
			return
		}
		switch strings.TrimSpace(splitLine[0]) {
		case "VERSION_MAJOR":
			VERSION_MAJOR = strings.TrimSpace(splitLine[1])
		case "VERSION_MINOR":
			VERSION_MINOR = strings.TrimSpace(splitLine[1])
		case "VERSION_PATCH":
			VERSION_PATCH = strings.TrimSpace(splitLine[1])
		}
	}
}

func fillBuildFromGit(localRepoPath string) {
	// equivalent to git rev-parse --short HEAD

	r, err := git.PlainOpen(localRepoPath)
	if err != nil {
		return // all of this is best-effort
	}

	h, err := r.ResolveRevision(plumbing.Revision("HEAD"))
	if err != nil {
		return
	}
	BUILD = h.String()[:7]
}

func init() {
	// If the core was built with go build directly instead of make.
	if VERSION_MAJOR == "0" &&
		VERSION_MINOR == "0" &&
		VERSION_PATCH == "0" &&
		BUILD == "" {
		exPath := getExecutableDir()
		basePath := filepath.Dir(exPath)
		versionFilePath := filepath.Join(basePath, "VERSION")

		if _, err := os.Stat(versionFilePath); err == nil {
			fillVersionFromVersionFile(versionFilePath)
		}

		fillBuildFromGit(basePath)
	}

	VERSION = strings.Join([]string{VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH}, ".")
	VERSION_SHORT = VERSION
	VERSION_BUILD = strings.Join([]string{VERSION, BUILD}, "-")
}

var ( // Acquired from -ldflags="-X=..." in Makefile
	VERSION_MAJOR = "0"
	VERSION_MINOR = "0"
	VERSION_PATCH = "0"
	BUILD         = ""
)

var (
	NAME             = "aliecs"
	PRETTY_SHORTNAME = "AliECS"
	PRETTY_FULLNAME  = "ALICE Experiment Control System"
	VERSION          string
	VERSION_SHORT    string
	VERSION_BUILD    string
)
