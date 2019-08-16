/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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

package repos

import (
	"errors"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"io/ioutil"
	"strings"
)

type Repo struct {
	HostingSite string
	User string
	RepoName string
	Revision string
	Default bool
}

func NewRepo(repoPath string) (*Repo, error) {

	revSlice := strings.Split(repoPath, "@")

	var repoUrlSlice []string
	var revision string
	if len(revSlice) == 2 { //revision specified
		repoUrlSlice = strings.Split(revSlice[0], "/")
		revision = revSlice[1]
	} else if len(revSlice) == 1{ //no revision specified
		repoUrlSlice = strings.Split(revSlice[0], "/")
		revision = "master"
	} else {
		return &Repo{}, errors.New("Repo path resolution failed")
	}

	if len(repoUrlSlice) < 3 {
		return &Repo{}, errors.New("Repo path resolution failed")
	}

	return &Repo{repoUrlSlice[0], repoUrlSlice[1],
		repoUrlSlice[2], revision, false}, nil
}

func (r *Repo) GetIdentifier() string {
	identifier := r.HostingSite + "/" + r.User + "/" + r.RepoName + "/"

	return identifier
}

func (r *Repo) getCloneDir() string {
	cloneDir := viper.GetString("repositoriesPath")
	if cloneDir[len(cloneDir)-1:] != "/" {
		cloneDir += "/"
	}

	cloneDir += r.HostingSite + "/" + r.User + "/" +r.RepoName

	return cloneDir
}

func (r *Repo) getCloneParentDirs() []string {
	cleanDir := viper.GetString("repositoriesPath")
	if cleanDir[len(cleanDir)-1:] != "/" {
		cleanDir += "/"
	}

	cleanDirUser := cleanDir +
		r.HostingSite + "/" +
		r.User

	cleanDirHostingSite := cleanDir +
		r.HostingSite

	ret := make([]string, 2)
	ret[0] = cleanDirUser
	ret[1] = cleanDirHostingSite
	return ret
}

func (r *Repo) getUrl() string {
	return "https://" +
		r.HostingSite + "/" +
		r.User 		  + "/" +
		r.RepoName	  + ".git"
}

func (r *Repo) getWorkflowDir() string {
	return r.getCloneDir() + "/workflows/"
}

func (r *Repo) ResolveTaskClassIdentifier(loadTaskClass string) (taskClassIdentifier string) {
	if !strings.Contains(loadTaskClass, "/") {
		taskClassIdentifier = r.HostingSite + "/" + r.User + "/" + r.RepoName + "/tasks/" + loadTaskClass
	} else {
		taskClassIdentifier = loadTaskClass
	}

	if !strings.Contains(loadTaskClass, "@") {
		if r.Revision == "" {
			taskClassIdentifier += "@master"
		} else {
			taskClassIdentifier += "@" + r.Revision
		}
	}

	return
}

func (r *Repo) checkoutRevision(revision string) error {
	if revision == "" {
		revision = r.Revision
	}

	ref, err := git.PlainOpen(r.getCloneDir())
	if err != nil {
		return err
	}

	w, err := ref.Worktree()
	if err != nil {
		return err
	}

	//Try remotely as a priority (branches) so that we don't check out old, dangling branch refs (e.g. master)
	newHash, err := ref.ResolveRevision(plumbing.Revision("origin/" + revision))
	if err != nil {
	 	//Try locally (tags + hashes)
		newHash, err = ref.ResolveRevision(plumbing.Revision(revision))
		if err != nil {
			return errors.New("checkoutRevision: " + err.Error())
		}
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash:  *newHash,
		Force: true,
	})
	if err != nil {
		return err
	}

	r.Revision = revision //Update repo revision
	return nil
}

func (r *Repo) refresh() error {

	ref, err := git.PlainOpen(r.getCloneDir())
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	err = ref.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force: true,
	})

	if err != nil && err.Error() != "already up-to-date" {
		return errors.New(err.Error() + ": " + r.GetIdentifier() + " | revision: " + r.Revision)
	}

	err = r.checkoutRevision("master")
	if err != nil {
		return err
	}

	return nil
}

func (r *Repo) getWorkflows() ([]string, error) {
	var workflows []string
	files, err := ioutil.ReadDir(r.getWorkflowDir())
	if err != nil {
		return workflows, err
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".yaml") { // Only return .yaml files
			workflows = append(workflows, strings.TrimSuffix(file.Name(), ".yaml"))
		}
	}
	return workflows, nil
}