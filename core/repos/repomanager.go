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
	ok, _ := rm.AddRepo(viper.GetString("defaultRepo"))
	if !ok {
		log.Fatal("Could not open default repo")
	}
	return &rm
}

func (manager *RepoManager) AddRepo(repoPath string) (bool, bool) { //TODO: Improve error handling?
	mutex.Lock()
	defer mutex.Unlock()

	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	repo, err := NewRepo(repoPath)

	if err != nil {
		return false, false
	}

	changedRevision := false
	_, exists := manager.repoList[repo.GetIdentifier()]
	if !exists { //Try to clone it
		token, err := ioutil.ReadFile("/home/kalexopo/git/o2-control-core.token") //TODO: Figure out AUTH

		auth := &http.BasicAuth {
			Username: "kalexopo",
			//Password: viper.GetString("repoToken"),
			Password: strings.TrimSuffix(string(token), "\n") ,
		}

		_, err = git.PlainClone(repo.GetCloneDir(), false, &git.CloneOptions{
			Auth:   auth,
			URL:    repo.GetUrl(),
			ReferenceName: plumbing.NewBranchReferenceName("master"),
		})

		if err != nil {
			if err.Error() == "repository already exists" { //Make sure master is checked out
				checkErr, _ := repo.CheckoutRevision("master")
				if checkErr != nil {
					return false, false
				}
				changedRevision = true
			} else {
				err = os.Remove(repo.GetCloneDir())
				//TODO: This doesn't help the persisting dir is the userdir which is unsafe to delete
				return false, false
			}
		}

		manager.repoList[repo.GetIdentifier()] = repo

		// Set default repo
		if len(manager.repoList) == 1 {
			manager.setDefaultRepo(repo)
		}
	}

	return true, changedRevision
}

func (manager *RepoManager) GetRepos() (repoList map[string]*Repo) {
	return manager.repoList
}

func (manager *RepoManager) RemoveRepo(repoPath string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	repo, exists := manager.repoList[repoPath]
	if exists {
		delete(manager.repoList, repoPath)
		if repo.Default && len(manager.repoList) > 0 {
			for _, newDefaultRepo := range manager.repoList { //Make a random repo default for now
				manager.setDefaultRepo(newDefaultRepo)
			}
		}
		return true
	} else {
		return false
	}
}

func (manager *RepoManager) RefreshRepos() (err error) { //TODO: One, more or all?
	//TODO: Fill me
	return
}

func (manager *RepoManager) GetWorkflow(workflowPath string)  (resolvedWorkflowPath string, workflowRepo *Repo, err error, changed bool) {

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
	if len(workflowInfo) == 1 { // Repo not specified
		workflowRepo = manager.GetDefaultRepo()
		workflowRepo.Revision = revision
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
		err = errors.New("Workflow comes from an unknown repo")
		return
	}

	// Make sure that HEAD is on the expected revision
	err, changed = workflowRepo.CheckoutRevision(workflowRepo.Revision)
	if err != nil {
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

func (manager *RepoManager) setDefaultRepo(repo *Repo){
	manager.defaultRepo = repo
	repo.Default = true
}


//---

/*func (repos RepoManager) SetRepoProperties(repoPath string, properties RepoProperties) (err error) {
	//TODO: Fill me
	return
}*/
