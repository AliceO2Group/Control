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
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
)

var (
	once sync.Once
	instance *RepoManager
	gitAuthUser = "kalexopo"
	gitAuthToken = "6RobMN4abw3kvpdz4iiQ"
)

func Instance() *RepoManager {
	once.Do(func() {
		instance = initializeRepos()
	})
	return instance
}

type RepoManager struct {
	repoList map[string]*Repo //I want this to work as a set
	defaultRepo *Repo
	mutex sync.Mutex
}

func initializeRepos() *RepoManager {
	rm := RepoManager{repoList: map[string]*Repo {}}
	err := rm.AddRepo(viper.GetString("defaultRepo"))
	if err != nil {
		log.Fatal("Could not open default repo: ", err)
	}

	//_ = rm.defaultRepo.checkoutRevision("v0.1.2")
	//_ = rm.defaultRepo.checkoutRevision("develop")
	//_ = rm.defaultRepo.checkoutRevision("87b65a2d89ce8155ccbc1bd593016f5ff4a3e3d7")
	//_ = rm.defaultRepo.checkoutRevision("87b65a")
	//_ = rm.RefreshRepos()

	return &rm
}

func (manager *RepoManager) AddRepo(repoPath string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	repo, err := NewRepo(repoPath)

	if err != nil {
		return err
	}

	_, exists := manager.repoList[repo.GetIdentifier()]
	if !exists { //Try to clone it

		auth := &http.BasicAuth {
			Username: gitAuthUser,
			Password: gitAuthToken,
		}

		_, err = git.PlainClone(repo.getCloneDir(), false, &git.CloneOptions{
			Auth:   auth,
			URL:    repo.getUrl(),
			ReferenceName: plumbing.NewBranchReferenceName(repo.Revision),
		})


		if err != nil {
			if err.Error() == "repository already exists" { //Make sure master is checked out
				checkoutErr := repo.checkoutRevision(repo.Revision)
				if checkoutErr != nil {
					return errors.New(err.Error() + " " + checkoutErr.Error())
				}
			} else {
				cleanErr := cleanCloneParentDirs(repo.getCloneParentDirs())
				if cleanErr != nil {
					return errors.New(err.Error() + " Failed to clean directories: " + cleanErr.Error())
				}
				return err
			}
		}

		manager.repoList[repo.GetIdentifier()] = repo

		// Set default repo
		if len(manager.repoList) == 1 {
			manager.setDefaultRepo(repo)
		}
	} else {
		return errors.New("Repo already present")
	}

	return nil
}

func cleanCloneParentDirs(parentDirs []string) error {
	for _, dir := range parentDirs {
		if empty, err := isEmpty(dir); empty {
			if err != nil {
				return err
			}

			err = os.Remove(dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func isEmpty(path string) (bool, error) { //TODO: Teo, should this be put somewhere like utils?
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer dir.Close() //Read-only we don't care about the return value

	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func (manager *RepoManager) GetRepos() (repoList map[string]*Repo) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	return manager.repoList
}

func (manager *RepoManager) RemoveRepoByIndex(index int) (ok bool, newDefaultRepo string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	keys := manager.GetOrderedRepolistKeys()

	for i, repoName := range keys {
		if i != index {
			continue
		}
		wasDefault := manager.repoList[repoName].Default
		delete(manager.repoList, repoName)
		// Set as default the repo sitting on top of the list
		if wasDefault && len(manager.repoList) > 0 {
			manager.setDefaultRepo(manager.repoList[manager.GetOrderedRepolistKeys()[0]]) //Keys have to be reparsed since there was a removal
			newDefaultRepo = keys[0]
		}
		return true, newDefaultRepo
	}

	return false, newDefaultRepo
}

func (manager *RepoManager) RemoveRepo(repoPath string) (ok bool) { //Unused
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	repo, exists := manager.repoList[repoPath]
	if exists {
		wasDefault := repo.Default
		delete(manager.repoList, repoPath)
		// Set as default the repo sitting on top of the list
		if wasDefault && len(manager.repoList) > 0 {
			manager.setDefaultRepo(manager.repoList[manager.GetOrderedRepolistKeys()[0]])
		}
		return true
	} else {
		return false
	}
}

func (manager *RepoManager) RefreshRepos() error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	for _, repo := range manager.repoList {

		err := repo.refresh()
		if err != nil {
			return errors.New("refresh repo for " + repo.GetIdentifier() + ":" + err.Error())
		}
	}

	return nil
}

func (manager *RepoManager) RefreshRepo(repoPath string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	repo := manager.repoList[repoPath]

	return repo.refresh()
}

func (manager *RepoManager) RefreshRepoByIndex(index int) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	keys := manager.GetOrderedRepolistKeys()

	for i, repoName := range keys {
		if i != index {
			continue
		}
		repo := manager.repoList[repoName]
		return repo.refresh()
	}

	return errors.New("RefreshRepoByIndex: repo not found for index: " + string(index))
}

func (manager *RepoManager) GetWorkflow(workflowPath string)  (resolvedWorkflowPath string, workflowRepo *Repo, err error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	// Get revision if present
	var revision string
	revSlice := strings.Split(workflowPath, "@")
	if len(revSlice) == 2 {
		workflowPath = revSlice[0]
		revision = revSlice[1]
	}

	// Resolve repo
	var workflowFile string
	workflowInfo := strings.Split(workflowPath, "workflows/")
	if len(workflowInfo) == 1 { // Repo not specified
		workflowRepo = manager.defaultRepo
		workflowFile = workflowInfo[0]
	} else if len(workflowInfo) == 2 { // Repo specified - try to find it
		workflowRepo= manager.repoList[workflowInfo[0]]
		if workflowRepo == nil {
			err = errors.New("Workflow comes from an unknown repo")
			return
		}

		workflowFile = workflowInfo[1]
	} else {
		err = errors.New("Workflow path resolution failed")
		return
	}

	if revision != "" { //If a revision has been specified, update the Repo
		workflowRepo.Revision = revision
	}

	// Make sure that HEAD is on the expected revision
	err = workflowRepo.checkoutRevision(revision)
	if err != nil {
		return
	}

	if !strings.HasSuffix(workflowFile, ".yaml") { //Add trailing ".yaml"
		workflowFile += ".yaml"
	}
	resolvedWorkflowPath = workflowRepo.getWorkflowDir() + workflowFile

	return
}

func (manager *RepoManager) setDefaultRepo(repo *Repo) {
	if manager.defaultRepo != nil {
		manager.defaultRepo.Default = false //Update old default repo
	}
	manager.defaultRepo = repo
	repo.Default = true
}

func (manager *RepoManager) UpdateDefaultRepoByIndex(index int) error {
	newDefaultRepo := manager.repoList[manager.GetOrderedRepolistKeys()[index]]
	if newDefaultRepo == nil {
		return errors.New("Repo not found")
	} else if newDefaultRepo == manager.defaultRepo {
		return errors.New(newDefaultRepo.GetIdentifier() + " is already the default repo")
	}

	manager.setDefaultRepo(newDefaultRepo)

	return nil
}

func (manager *RepoManager) UpdateDefaultRepo(repoPath string) error { //unused
	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	newDefaultRepo := manager.repoList[repoPath]
	if newDefaultRepo == nil {
		return errors.New("Repo not found")
	} else if newDefaultRepo == manager.defaultRepo {
		return errors.New(newDefaultRepo.GetIdentifier() + " is already the default repo")
	}

	manager.setDefaultRepo(newDefaultRepo)

	return nil
}

func (manager *RepoManager) EnsureReposPresent(taskClassesRequired []string) (err error) {
	reposRequired := make(map[Repo]bool)
	for _, taskClass := range taskClassesRequired {
		var newRepo *Repo
		newRepo, err = NewRepo(taskClass)
		if err != nil {
			return
		}
		reposRequired[*newRepo] = true
	}

	// Make sure that the relevant repos are present and checked out on the expected revision
	for repo  := range reposRequired {
		existingRepo, ok := manager.repoList[repo.GetIdentifier()]
		if !ok {
			err = manager.AddRepo(repo.GetIdentifier())
			if err != nil {
				return
			}
		} else {
			if existingRepo.Revision != repo.Revision {
				err = existingRepo.checkoutRevision(repo.Revision)
				if err != nil {
					return
				}
			}
		}
	}

	return
}

func (manager *RepoManager) GetOrderedRepolistKeys() []string {
	// Ensure alphabetical order of repos in output
	var keys []string
	for key := range manager.repoList {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
