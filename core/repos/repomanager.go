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
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/core/confsys"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	refPrefix       = "refs/"
	refTagPrefix    = refPrefix + "tags/"
	refRemotePrefix = refPrefix + "remotes/origin/"
)

var log = logger.New(logrus.StandardLogger(),"repos")

var (
	once sync.Once
	instance *RepoManager
)

func Instance(service *confsys.Service) *RepoManager {
	once.Do(func() {
		instance = initializeRepos(service)
	})
	return instance
}

type RepoManager struct {
	repoList map[string]*Repo
	defaultRepo *Repo
	mutex sync.Mutex
	cService *confsys.Service
}

func initializeRepos(service *confsys.Service) *RepoManager {
	rm := RepoManager{repoList: map[string]*Repo {}}
	rm.cService = service

	// Get default repo
	defaultRepo, err := rm.cService.GetDefaultRepo()
	if err != nil {
		log.Warning("Failed to parse default_repo from backend")
		defaultRepo = viper.GetString("defaultRepo")
	}

	// Add default repo
	err = rm.AddRepo(defaultRepo)
	if err != nil {
		log.Fatal("Could not open default repo: ", err)
	}

	// Discover & add repos from filesystem
	var discoveredRepos []string
	discoveredRepos, err = rm.discoverRepos()
	if err != nil {
		log.Warning("Failed on discovery of existing repos: ", err)
	}

	for _, repo := range discoveredRepos {
		err = rm.AddRepo(repo)
		if err != nil && err.Error() != "Repo already present" { //Skip error for default repo
			log.Warning("Failed to add persistent repo: ", repo, " | ", err)
		}
	}

	return &rm
}

func (manager *RepoManager)  discoverRepos() (repos []string, err error){
	var hostingSites []string
	var usernames []string
	var someRepos []string

	hostingSites, err = filepath.Glob(viper.GetString("RepositoriesPath") + "*")
	if err != nil {
		return
	}

	for _, hostingSite := range hostingSites {
		usernames, err = filepath.Glob(hostingSite + "/*")
		if err != nil {
			return
		}
		for _, username := range usernames {
			someRepos, err = filepath.Glob(username + "/*")
			if err != nil {
				return
			}

			for _, repo := range someRepos { //sanitize path
				repo = strings.TrimPrefix(repo, viper.GetString("RepositoriesPath"))
				repos = append(repos, repo)
			}
		}
	}

	return
}

func (manager *RepoManager) AddRepo(repoPath string) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	utils.EnsureTrailingSlash(&repoPath)

	repo, err := NewRepo(repoPath)

	if err != nil {
		return err
	}

	_, exists := manager.repoList[repo.GetIdentifier()]
	if !exists { //Try to clone it


		_, err = git.PlainClone(repo.getCloneDir(), false, &git.CloneOptions{
			URL:    repo.getUrl(),
			ReferenceName: plumbing.NewBranchReferenceName(repo.Revision),
		})


		if err != nil {
			if err == git.ErrRepositoryAlreadyExists { //Make sure master is checked out
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
		if empty, err := utils.IsDirEmpty(dir); empty {
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

func (manager *RepoManager) GetAllRepos() (repoList map[string]*Repo) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	return manager.repoList
}

func (manager *RepoManager) getRepos(repoPattern string) (repoList map[string]*Repo) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if repoPattern == "*" { // Skip unnecessary pattern matching
		return manager.repoList
	}

	matchedRepoList := make(map[string]*Repo)
	g := glob.MustCompile(repoPattern)

	for _, repo := range manager.repoList {
		if g.Match(repo.GetIdentifier()) {
			matchedRepoList[repo.GetIdentifier()] = repo
		}
	}

	return matchedRepoList
}

func (manager *RepoManager) getRepoByIndex(index int) (*Repo, error) {
	keys := manager.GetOrderedRepolistKeys()
	if len(keys) - 1 >= index { // Verify that index is not out of bounds
		return manager.repoList[keys[index]], nil
	} else {
		return nil, errors.New("getRepoByIndex: repo not found for index :" + strconv.Itoa(index))
	}
}

func (manager *RepoManager) RemoveRepoByIndex(index int) (string, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	newDefaultRepoString := ""

	repo, err := manager.getRepoByIndex(index)
	if err != nil {
		return "", err
	}

	wasDefault := repo.Default

	_ = os.RemoveAll(repo.getCloneDir()) // Try, but don't crash if we fail

	delete(manager.repoList, repo.GetIdentifier())
	// Set as default the repo sitting on top of the list
	if wasDefault && len(manager.repoList) > 0 {
		newDefaultRepo, err := manager.getRepoByIndex(0)
		if err != nil {
			return newDefaultRepoString, err
		}
		manager.setDefaultRepo(newDefaultRepo)
		newDefaultRepoString = newDefaultRepo.GetIdentifier()
	} else if len(manager.repoList) == 0 {
		err := manager.cService.NewDefaultRepo(viper.GetString("defaultRepo"))
		if err != nil {
			log.Warning("Failed to update default_repo backend")
		}
	}
	return newDefaultRepoString, nil
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

	utils.EnsureTrailingSlash(&repoPath)

	repo := manager.repoList[repoPath]

	return repo.refresh()
}

func (manager *RepoManager) RefreshRepoByIndex(index int) error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	repo, err := manager.getRepoByIndex(index)
	if err != nil {
		return err
	}

	return repo.refresh()
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
	err = workflowRepo.checkoutRevision(workflowRepo.Revision)
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
		manager.defaultRepo.Default = false // Update old default repo
	}

	// Update default_repo backend
	err := manager.cService.NewDefaultRepo(repo.GetIdentifier())
	if err != nil {
		log.Warning("Failed to update default_repo backend: ", err)
	}

	manager.defaultRepo = repo
	repo.Default = true
}

func (manager *RepoManager) UpdateDefaultRepoByIndex(index int) error {
	orderedRepoList := manager.GetOrderedRepolistKeys()
	if index >= len(orderedRepoList) {
		return errors.New("Default repo index out of range")
	}
	newDefaultRepo := manager.repoList[orderedRepoList[index]]
	if newDefaultRepo == nil {
		return errors.New("Repo not found")
	} else if newDefaultRepo == manager.defaultRepo {
		return errors.New(newDefaultRepo.GetIdentifier() + " is already the default repo")
	}

	manager.setDefaultRepo(newDefaultRepo)

	return nil
}

func (manager *RepoManager) UpdateDefaultRepo(repoPath string) error { //unused
	utils.EnsureTrailingSlash(&repoPath)

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

type RepoKey string
type RevisionKey string
type Template string
type Templates []Template
type TemplatesByRevision map[RevisionKey]Templates
type TemplatesByRepo map[RepoKey]TemplatesByRevision

// Returns a map of templates: repo -> revision -> []templates
func (manager *RepoManager) GetWorkflowTemplates(repoPattern string, revisionPattern string, allBranches bool, allTags bool) (TemplatesByRepo, int, error) {
	templateList := make(TemplatesByRepo)
	numTemplates := 0

	if repoPattern == "" { // all repos if unspecified
		repoPattern = "*"
	}

	if allBranches || allTags {
		if revisionPattern != "" { // If the revision pattern is specified and an all{Branches,Tags} flag is used return error
			return nil, 0, errors.New("cannot use all{Branches,Tags} with a revision specified")
		}
	} else if revisionPattern == "" { // master if unspecified
		revisionPattern = "master"
	}

	// Prepare the gitRefs slice which will filter the git references
	var gitRefs []string
	if !allBranches && !allTags {
		gitRefs = append(gitRefs, refRemotePrefix, refTagPrefix)
	} else {
		revisionPattern = "*"
		if allBranches {
			gitRefs = append(gitRefs, refRemotePrefix)
		}

		if allTags {
			gitRefs = append(gitRefs, refTagPrefix)
		}
	}

	// Build list of repos to iterate through by pattern or by index(single repo, if pattern is an int)
	repos := make(map[string]*Repo)
	if repoIndex, err := strconv.Atoi(repoPattern); err == nil {
		repo, err := manager.getRepoByIndex(repoIndex)
		if err != nil {
			return nil, 0, err
		}
		repos[repo.GetIdentifier()] = repo
	} else {
		repos = manager.getRepos(repoPattern)
	}

	for _, repo := range repos {
		// For every repo get the templates for the revisions matching the revisionPattern; gitRefs to filter tags and/or branches
		templates, err := repo.getWorkflows(revisionPattern, gitRefs)
		if err != nil {
			return nil, 0, err
		}
		templateList[RepoKey(repo.GetIdentifier())] = templates

		for _, revTemplate := range templates {
			numTemplates += len(revTemplate) // numTemplates is needed for protobuf to know the number of messages to go through
		}

	}

	return templateList, numTemplates, nil
}
