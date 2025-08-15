/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Kostas Alexopoulos <kostas.alexopoulos@cern.ch>
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

// Package repos provides repository management functionality for accessing
// and synchronizing Git repositories containing workflow templates and configurations.
package repos

import (
	"errors"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/viper"
	ssh2 "golang.org/x/crypto/ssh"
	"net/url"
	"os/exec"
	"path"
)

type sshRepo struct {
	Repo
}

func (r *sshRepo) getUri() string {
	u, _ := url.Parse(r.GetProtocol() + "://")
	u.Path = path.Join(u.Path, "git@"+r.HostingSite)

	u.Path = path.Join(u.Path,
		r.Path,
		r.RepoName)

	uri := u.String()

	return uri + ".git"
}

func (r *sshRepo) GetProtocol() string {
	return "ssh"
}

func (r *sshRepo) refresh() error {
	ref, err := git.PlainOpen(r.GetCloneDir())
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	auth, err := ssh.NewPublicKeysFromFile("git", viper.GetString("reposSshKey"), "")
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	// Disable strict host checking without which may block the fetch op without manual intervention
	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	// clean the repo before doing anything
	// this removes the untracked JIT-produced tasks and workflows
	clnCmd := exec.Command("git", "-C", r.GetCloneDir(), "clean", "-f")
	err = clnCmd.Run()
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	err = ref.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
		Auth:       auth,
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.New(err.Error() + ": " + r.GetIdentifier() + " | revision: " + r.Revision)
	}

	// gather revisions on update or if empty
	if err != git.NoErrAlreadyUpToDate || r.Revisions == nil {
		err = r.gatherRevisions(ref)
		if err != nil {
			return err
		}
	}

	// populate workflows on update or if empty
	if err != git.NoErrAlreadyUpToDate || len(templatesCache) == 0 {
		err = r.populateWorkflows(r.GetDefaultRevision(), true)
		if err != nil {
			return err
		}
	}

	return nil
}

/*func (r *sshRepo) GetDplCommand(dplCommandUri string) (string, error) {
	dplCommandPath := filepath.Join(r.GetCloneDir(), jitScriptsDir, dplCommandUri)
	dplCommandPayload, err := os.ReadFile(dplCommandPath)
	if err != nil {
		return "", err
	}
	return string(dplCommandPayload), nil
}*/
