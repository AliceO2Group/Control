package repos

import (
	"errors"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

var (
	once sync.Once
	instance *RepoManager
	mutex sync.Mutex // move to struct
)

func Instance() *RepoManager {
	once.Do(func() {
		instance = initializeRepos()
	})
	return instance
}

type RepoManager struct {
	repoList map[string]bool //I want this to work as a set
}

func initializeRepos() *RepoManager {
	return &RepoManager{repoList: map[string]bool {viper.GetString("defaultRepo"): true}}
}

func (repos *RepoManager) populateRepoList() (err error) {
	//TODO: Go through the dir and get whatever git info you can
	return
}

func (repos *RepoManager) AddRepo(repoPath string) bool { //TODO: Add smarter error handling?
	mutex.Lock()
	defer mutex.Unlock()

	if repoPath[len(repoPath)-1:] != "/" { //Add trailing '/'
		repoPath += "/"
	}

	_, exists := repos.repoList[repoPath]
	if !exists {
		repos.repoList[repoPath] = true
		return true
	} else {
		return false
	}
}

func (repos *RepoManager) GetRepos() (repoList map[string]bool, err error) {
	return repos.repoList, nil
}

func (repos *RepoManager) RemoveRepo(repoPath string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if repoPath[len(repoPath)-1:] != "/" { //Add trailing '/'
		repoPath += "/"
	}

	_, exists := repos.repoList[repoPath]
	if exists {
		delete(repos.repoList, repoPath)
		return true
	} else {
		return false
	}
}

func (repos *RepoManager) RefreshRepos() (err error) { //TODO: One, more or all?
	//TODO: Fill me
	return
}

func (repos *RepoManager) GetWorkflow(workflowPath string)  (resolvedWorkflowPath string, workflowRepo string, err error) {
	workflowInfo := strings.Split(workflowPath, "workflows/")
	workflowRepo = workflowInfo[0]
	if workflowRepo == "" {
		workflowRepo = viper.GetString("defaultRepo")
	}
	//workflowDir := workflowInfo[1]

	// TODO: Check that the repo is already available
	_, exists := repos.repoList[workflowRepo]
	if !exists {
		return "", "", errors.New("workflow comes from an unknown repo")
	}

	// Resolve actual workflow path
	if strings.Contains(workflowPath, "/") {
		resolvedWorkflowPath = viper.GetString("repositoriesUri") +  workflowPath + ".yaml"
	} else {
		resolvedWorkflowPath = viper.GetString("repositoriesUri") + viper.GetString("defaultRepo")  +
		"workflows/" + workflowPath + ".yaml"
	}

	return resolvedWorkflowPath, workflowRepo, nil
	//TODO: improve error handling
}


//---


type RepoProperties struct {
	propA string
	propB string
}

func (repos RepoManager) SetRepoProperties(repoPath string, properties RepoProperties) (err error) {
	//TODO: Fill me
	return
}

func TraverseTasksPath() (err error) {
	return
}

func TraverseWorkflowsPath() (err error) {
	return
}