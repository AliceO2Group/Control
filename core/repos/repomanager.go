package repos

import (
	"errors"
	"github.com/spf13/viper"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"io/ioutil"
	"log"
	"os"
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
	repoList map[string]*Repo //I want this to work as a set
	defaultRepo *Repo
}

func initializeRepos() *RepoManager {
	rm := RepoManager{repoList: map[string]*Repo {}}
	ok := rm.AddRepo(viper.GetString("defaultRepo"))
	if !ok {
		log.Fatal("Could not open default repo")
	}
	return &rm
}

func (manager *RepoManager) AddRepo(repoPath string) bool { //TODO: Improve error handling?
	mutex.Lock()
	defer mutex.Unlock()

	repo, err := NewRepo(repoPath)

	if err != nil {
		return false //TODO: error handling
	}

	_, exists := manager.repoList[repo.GetIdentifier()]
	if !exists { //Try to clone it
		token, err := ioutil.ReadFile("/home/kalexopo/git/o2-control-core.token") //TODO: Figure out AUTH

		_, err = git.PlainClone(repo.GetCloneDir(), false, &git.CloneOptions{
			Auth: &http.BasicAuth {
				Username: "kalexopo",
				//Password: viper.GetString("repoToken"),
				Password: strings.TrimSuffix(string(token), "\n") ,
			},
			URL:    repo.GetUrl(),
			ReferenceName: plumbing.NewBranchReferenceName(repo.Revision),
			//Progress: os.Stdout,
		})

		if err != nil && (err.Error() != "repository already exists") { //We coulnd't add the repo
			err = os.Remove(repo.GetCloneDir())
			//TODO: This doesn't help the persisting dir is the userdir which is unsafe to delete
			return false
		}

		manager.repoList[repo.GetIdentifier()] = repo

		// Set default repo
		if len(manager.repoList) == 1 {
			manager.defaultRepo = repo
		}
	}

	return true
}

func (manager *RepoManager) GetRepos() (repoList map[string]*Repo) {
	return manager.repoList
}

func (manager *RepoManager) RemoveRepo(repoPath string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	_, exists := manager.repoList[repoPath]
	if exists {
		delete(manager.repoList, repoPath)
		return true
	} else {
		return false
	}
}

func (manager *RepoManager) RefreshRepos() (err error) { //TODO: One, more or all?
	//TODO: Fill me
	return
}

func (manager *RepoManager) GetWorkflow(workflowPath string)  (resolvedWorkflowPath string, workflowRepo *Repo, err error) { //TODO: Move to Repo?

	// Get revision if present
	var revision string
	revSlice := strings.Split(workflowPath, "@")
	if len(revSlice) == 2 {
		workflowPath = revSlice[0]
		revision = revSlice[1]
	}

	// Get repo
	var workflowFile string
	workflowInfo := strings.Split(workflowPath, "/workflows/")
	if len(workflowInfo) == 1 { // Repo not specified
		workflowRepo = manager.GetDefaultRepo()
		workflowFile = workflowInfo[0]
	} else if len(workflowInfo) == 2 { // Repo specified
		workflowRepo, err = NewRepo(workflowInfo[0])
		if err != nil {
			return
		}

		workflowFile = workflowInfo[1]
		if revision != "" {
			workflowRepo.Revision = revision
		}
	} else {
		err = errors.New("Workflow path resolution failed")
		return
	}


	// Check that the repo is already available
	_, exists := manager.repoList[workflowRepo.GetIdentifier()]
	if !exists {
		err = errors.New("Workflow comes from an unknow repo")
		return
	}

	if !strings.HasSuffix(workflowFile, ".yaml") { //Add trailing ".yaml"
		workflowFile += ".yaml"
	}
	resolvedWorkflowPath = workflowRepo.GetWorkflowDir() + workflowFile

	return
}

func (manager *RepoManager) GetDefaultRepo() *Repo { //TODO: To be reworked with prioritites
	return manager.defaultRepo
}

func (manager *RepoManager) SetDefaultRepo(repo *Repo){
	manager.defaultRepo = repo
}


//---

/*func (repos RepoManager) SetRepoProperties(repoPath string, properties RepoProperties) (err error) {
	//TODO: Fill me
	return
}*/
