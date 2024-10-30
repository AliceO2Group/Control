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
	"fmt"
	"github.com/AliceO2Group/Control/common/logger"
	"github.com/AliceO2Group/Control/common/utils"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/gobwas/glob"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	ssh2 "golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
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

var log = logger.New(logrus.StandardLogger(), "repos")

var (
	once     sync.Once
	instance *RepoManager
)

func Instance(service configuration.Service) *RepoManager {
	once.Do(func() {
		instance = initializeRepos(service)
	})
	return instance
}

type RepoManager struct {
	repoList         map[string]iRepo
	defaultRepo      iRepo
	defaultRevision  string
	defaultRevisions map[string]string
	mutex            sync.Mutex
	cService         configuration.Service
	rService         *RepoService
}

func initializeRepos(service configuration.Service) *RepoManager {
	rm := RepoManager{repoList: map[string]iRepo{}}
	rm.cService = service
	rm.rService = &RepoService{Svc: service}

	var err error
	// Get global default revision
	rm.defaultRevision, err = rm.rService.GetDefaultRevision()
	if err != nil || rm.defaultRevision == "" {
		log.Debug("Failed to parse default_revision from backend")
		rm.defaultRevision = viper.GetString("globalDefaultRevision")
	} else {
		viper.Set("globalDefaultRevision", rm.defaultRevision)
	}

	// Get default revisions
	revsMap, err := rm.rService.GetRepoDefaultRevisions()
	if err != nil {
		log.Debug("Failed to parse default_revisions from backend")
		rm.defaultRevisions = make(map[string]string)
	} else {
		rm.defaultRevisions = revsMap
	}

	// Get default repo
	defaultRepo, err := rm.rService.GetDefaultRepo()
	if err != nil || defaultRepo == "" {
		log.Warning("Failed to parse default_repo from backend")
		defaultRepo = viper.GetString("defaultRepo")
	}

	_, _, err = rm.AddRepo(defaultRepo, rm.defaultRevisions[defaultRepo])
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
		_, _, err = rm.AddRepo(repo, rm.defaultRevisions[repo])
		if err != nil && err.Error() != "Repo already present" { //Skip error for default repo
			log.Warning("Failed to add persistent repo: ", repo, " | ", err)
		}
	}

	// Update all repos
	err = rm.RefreshRepos()
	if err != nil {
		log.Warning("Could not refresh repos: ", err)
	}

	return &rm
}

func (manager *RepoManager) discoverRepos() (repos []string, err error) {
	var hostingSites []string

	hostingSites, err = filepath.Glob(filepath.Join(manager.rService.GetReposPath(), "*"))
	if err != nil {
		return
	}

	// A path is valid in this context if it is a dir and isn't hidden (e.g. .git)
	isValidPath := func(path string) bool {
		fileInfo, err := os.Stat(path)
		if err == nil && fileInfo.IsDir() && fileInfo.Name()[0:1] != "." {
			return true
		}
		return false
	}

	for _, hostingSite := range hostingSites {
		// Get rid of invalid paths
		if !isValidPath(hostingSite) {
			continue
		}

		// find .git dir
		// everything above is "path", dir containing ".git" is repo
		err = filepath.Walk(hostingSite, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && info.Name() == ".git" {
				// hack to "sanitize" the path
				// i.e. don't include repos path
				// don't include trailing "/.git"
				repos = append(repos, strings.TrimPrefix(strings.TrimSuffix(path, "/.git"),
					manager.rService.GetReposPath()+"/"))
			}

			return nil
		})

		if err != nil {
			return
		}
	}

	return
}

func (manager *RepoManager) checkAndSetDefaultRevision(repo iRepo) error {
	// Decide if requested revision = defaultRevision -> if yes lock them together
	// We do this because, in the case where the revision was not specified, it would fall back to the default revision
	// As a result, subsequent changes to the default revision, within the scope of this function, should be followed by revision
	revIsDefault := false
	if repo.GetDefaultRevision() == repo.getRevision() {
		revIsDefault = true
	}

	// Check that the defaultRevision is valid, i.e. can be checked out
	var prefixes []string
	prefixes = append(prefixes, refRemotePrefix, refTagPrefix)
	matchedRevs, err := repo.getRevisions(repo.GetDefaultRevision(), prefixes)
	if err != nil {
		return err
	} else if len(matchedRevs) == 0 {
		log.Warning("Default revision " + repo.GetDefaultRevision() + " invalid for " + repo.GetIdentifier())
		if repo.GetDefaultRevision() != manager.defaultRevision {
			log.Warning("Defaulting to global default revision: " + manager.defaultRevision)
			repo.setDefaultRevision(manager.defaultRevision)
			matchedRevs, err = repo.getRevisions(repo.GetDefaultRevision(), prefixes)
			if err != nil {
				return err
			} else if len(matchedRevs) == 0 {
				log.Warning("Global default revision " + repo.GetDefaultRevision() + " invalid for " + repo.GetIdentifier())

			} else {
				// if the revision was the default revision, make sure we now follow the updated default revision
				if revIsDefault {
					repo.setRevision(repo.GetDefaultRevision())
				}
				return nil
			}
		}

		// No revisions matched, fall back to master or main
		matchedRevs, err = repo.getRevisions("master", prefixes)
		if err == nil && len(matchedRevs) != 0 {
			log.Warning("Defaulting to master")
			repo.setDefaultRevision("master")
			return nil
		}

		matchedRevs, err = repo.getRevisions("main", prefixes)
		if err == nil && len(matchedRevs) != 0 {
			log.Warning("Defaulting to main")
			repo.setDefaultRevision("main")
			return nil
		}

		log.Warning("Cannot fall back to any default revision for ", repo.GetIdentifier())
		return fmt.Errorf("cannot fall back to any default revision for %s", repo.GetIdentifier())
	}

	return nil
}

func (manager *RepoManager) AddRepo(repoPath string, defaultRevision string) (string, bool, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if defaultRevision == "" {
		defaultRevision = manager.defaultRevision
	}

	repo, err := newRepo(repoPath, defaultRevision, manager.rService.GetReposPath())

	if err != nil {
		return "", false, err
	}

	_, exists := manager.repoList[repo.GetIdentifier()]
	if !exists { //Try to clone it

		var gitRepo *git.Repository
		if repo.GetProtocol() == "local" { // local repo needs only be opened
			// We first need to make sure that the local repo is accessible, if not `git.PlainOpen` will crash us with a segfault
			// It should also be writable for JIT
			err := unix.Access(repo.GetCloneDir(), unix.W_OK)
			if err != nil {
				return "", false, err
			}

			// TODO: persistent local repos?
			gitRepo, err = git.PlainOpen(repo.GetCloneDir())
		} else { // https and ssh repos need to be cloned
			var co git.CloneOptions

			// prepare clone options for ssh authorization
			if repo.GetProtocol() == "ssh" {
				auth, err := ssh.NewPublicKeysFromFile("git", viper.GetString("reposSshKey"), "")
				if err != nil {
					return "", false, err
				}

				// Disable strict host checking which may block the clone op without manual intervention
				auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
				co.Auth = auth
			}

			co.URL = repo.getUri()
			co.NoCheckout = true

			gitRepo, err = git.PlainClone(repo.GetCloneDir(), false, &co)
		}

		if err != nil && err != git.ErrRepositoryAlreadyExists {
			// Something went wrong, clean up
			cleanErr := cleanCloneParentDirs(repo.getCloneParentDirs())
			if cleanErr != nil {
				return "", false, errors.New(err.Error() + " Failed to clean directories: " + cleanErr.Error())
			}
			return "", false, err
		}

		// Make sure the repo is up to date and its structs are populated
		err = repo.refresh()
		if err != nil {
			return "", false, err
		}

		if gitRepo != nil {
			// Update git config to be case insensitive
			var gitConfig *config.Config
			gitConfig, err = gitRepo.Config()
			if err != nil {
				return "", false, errors.New(err.Error() + " Failed to get git config")
			}

			gitConfig.Raw.AddOption("core", "", "ignorecase", "true")
			err = gitRepo.SetConfig(gitConfig)
			if err != nil {
				return "", false, errors.New(err.Error() + " Failed to update git config")
			}
		}

		// Check that the repo's default revision is valid, update if not
		err = manager.checkAndSetDefaultRevision(repo)
		if err != nil {
			return "", false, err
		}

		manager.repoList[repo.GetIdentifier()] = repo

		// Set default repo
		if len(manager.repoList) == 1 {
			manager.setDefaultRepo(repo)
		}
	} else {
		return "", false, errors.New("Repo already present")
	}

	// Update default revisions
	manager.defaultRevisions[repo.GetIdentifier()] = repo.GetDefaultRevision()
	err = manager.rService.SetRepoDefaultRevisions(manager.defaultRevisions)
	if err != nil {
		return "", false, err
	}

	return repo.GetDefaultRevision(), repo.GetDefaultRevision() == manager.defaultRevision, nil
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

func (manager *RepoManager) GetAllRepos() (repoList map[string]iRepo) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	return manager.repoList
}

func (manager *RepoManager) getRepos(repoPattern string) (repoList map[string]iRepo) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if repoPattern == "*" { // Skip unnecessary pattern matching
		return manager.repoList
	}

	matchedRepoList := make(map[string]iRepo)
	g := glob.MustCompile(repoPattern)

	for _, repo := range manager.repoList {
		if g.Match(repo.GetIdentifier()) {
			matchedRepoList[repo.GetIdentifier()] = repo
		}
	}

	return matchedRepoList
}

func (manager *RepoManager) getRepoByIndex(index int) (iRepo, error) {
	keys := manager.GetOrderedRepolistKeys()
	if len(keys)-1 >= index { // Verify that index is not out of bounds
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

	wasDefault := repo.IsDefault()

	_ = os.RemoveAll(repo.GetCloneDir())                // Try, but don't crash if we fail
	_ = cleanCloneParentDirs(repo.getCloneParentDirs()) // Try, but don't crash if we fail

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
		err := manager.rService.NewDefaultRepo(viper.GetString("defaultRepo"))
		if err != nil {
			log.Warning("Failed to update default_repo backend")
		}
	}

	// Update default revisions
	delete(manager.defaultRevisions, repo.GetIdentifier())
	err = manager.rService.SetRepoDefaultRevisions(manager.defaultRevisions)
	if err != nil {
		return "", err
	}

	return newDefaultRepoString, nil
}

func (manager *RepoManager) RefreshRepos() error {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	for _, repo := range manager.repoList {

		manager.defaultRevisions[repo.GetIdentifier()] = repo.GetDefaultRevision()
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

func (manager *RepoManager) GetWorkflow(workflowPath string) (resolvedWorkflowPath string, workflowRepo IRepo, err error) {
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
	workflowInfo := strings.Split(workflowPath, "/workflows/")
	var wfRepo iRepo
	if len(workflowInfo) == 1 { // Repo not specified
		wfRepo = manager.defaultRepo
		workflowFile = workflowInfo[0]
	} else if len(workflowInfo) == 2 { // Repo specified - try to find it
		wfRepo = manager.repoList[workflowInfo[0]]
		if wfRepo == nil {
			err = errors.New("Workflow comes from an unknown repo")
			return
		}

		workflowFile = workflowInfo[1]
	} else {
		err = errors.New("Workflow path resolution failed")
		return
	}

	if revision != "" { // If a revision has been specified, update the Repo
		wfRepo.setRevision(revision)
	} else { // Otherwise use the default revision of the repo
		wfRepo.setDefaultRevision(wfRepo.GetDefaultRevision())
	}

	// Make sure that HEAD is on the expected revision
	err = wfRepo.checkoutRevision(wfRepo.getRevision())
	if err != nil {
		return
	}

	if !strings.HasSuffix(workflowFile, ".yaml") { //Add trailing ".yaml"
		workflowFile += ".yaml"
	}
	resolvedWorkflowPath = filepath.Join(wfRepo.getWorkflowDir(), workflowFile)

	workflowRepo = IRepo(wfRepo)

	return
}

func (manager *RepoManager) setDefaultRepo(repo iRepo) {
	if manager.defaultRepo != nil {
		manager.defaultRepo.setDefault(false) // Update old default repo
	}

	// Update default_repo backend
	err := manager.rService.NewDefaultRepo(repo.GetIdentifier())
	if err != nil {
		log.Warning("Failed to update default_repo backend: ", err)
	}

	manager.defaultRepo = repo
	repo.setDefault(true)
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
	newDefaultRepo := manager.repoList[repoPath]
	if newDefaultRepo == nil {
		return errors.New("Repo not found")
	} else if newDefaultRepo == manager.defaultRepo {
		return errors.New(newDefaultRepo.GetIdentifier() + " is already the default repo")
	}

	manager.setDefaultRepo(newDefaultRepo)

	return nil
}

func (manager *RepoManager) SetGlobalDefaultRevision(revision string) error {
	// Update default_revision backend
	err := manager.rService.NewDefaultRevision(revision)
	if err != nil {
		log.Warning("Failed to update default_revision backend: ", err)
		return err
	}

	viper.Set("globalDefaultRevision", revision)
	manager.defaultRevision = revision
	return nil
}

func (manager *RepoManager) UpdateDefaultRevisionByIndex(index int, revision string) (string, error) {
	orderedRepoList := manager.GetOrderedRepolistKeys()
	if index >= len(orderedRepoList) {
		return "", errors.New("repository index out of range")
	}
	repo := manager.repoList[orderedRepoList[index]]

	info, err := repo.updateDefaultRevision(revision)
	if err != nil {
		return info, errors.New("Could not update default revision for " + repo.GetIdentifier() + " : " + err.Error())
	}

	// Update default revisions
	manager.defaultRevisions[repo.GetIdentifier()] = repo.GetDefaultRevision()
	err = manager.rService.SetRepoDefaultRevisions(manager.defaultRevisions)
	if err != nil {
		return "", err
	}

	return "", nil
}

func (manager *RepoManager) EnsureReposPresent(taskClassesRequired []string) (err error) {
	reposRequired := make(map[string]iRepo)
	for _, taskClass := range taskClassesRequired {
		var taskRepo iRepo
		taskRepo, err = newRepo(taskClass, manager.defaultRevision, manager.rService.GetReposPath())
		if err != nil {
			return
		}
		reposRequired[taskRepo.GetIdentifier()] = taskRepo
	}

	// Make sure that the relevant repos are present and checked out on the expected revision
	for _, repo := range reposRequired {
		existingRepo, ok := manager.repoList[repo.GetIdentifier()]
		if !ok {
			_, _, err = manager.AddRepo(repo.GetIdentifier(), repo.GetDefaultRevision())
			if err != nil {
				return
			}
		} else {
			if existingRepo.getRevision() != repo.getRevision() {
				err = existingRepo.checkoutRevision(repo.getRevision())
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
type Template struct {
	Name        string
	Description string
	Public      bool
	VarInfo     VarSpecMap
}
type Templates []Template
type TemplatesByRevision map[RevisionKey]Templates
type TemplatesByRepo map[RepoKey]TemplatesByRevision

// GetWorkflowTemplates Returns a map of templates: repo -> revision -> []templates
func (manager *RepoManager) GetWorkflowTemplates(repoPattern string, revisionPattern string, allBranches bool, allTags bool, allWorkflows bool) (TemplatesByRepo, int, error) {
	templateList := make(TemplatesByRepo)
	numTemplates := 0

	if repoPattern == "" { // all repos if unspecified
		repoPattern = "*"
	}

	if allBranches || allTags {
		if revisionPattern != "" { // If the revision pattern is specified and an all{Branches,Tags} flag is used return error
			return nil, 0, errors.New("cannot use all{Branches,Tags} with a revision specified")
		}
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
	repos := make(map[string]iRepo)
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
		var templates TemplatesByRevision
		var err error
		if revisionPattern == "" { // If the revision pattern is empty, use the default revision
			templates, err = repo.getWorkflows(repo.GetDefaultRevision(), gitRefs, allWorkflows)
		} else {
			templates, err = repo.getWorkflows(revisionPattern, gitRefs, allWorkflows)
		}

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

func (manager *RepoManager) GetDefaultRevision() string {
	return manager.defaultRevision
}

func (manager *RepoManager) GetReposPath() string {
	return manager.rService.GetReposPath()
}
