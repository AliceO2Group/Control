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
	"strings"
	"sync"
)

var (
	once sync.Once
	instance *RepoManager
	mutex sync.Mutex // move to struct
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
}

func initializeRepos() *RepoManager {
	rm := RepoManager{repoList: map[string]*Repo {}}
	err := rm.AddRepo(viper.GetString("defaultRepo"))
	if err != nil {
		log.Fatal("Could not open default repo: ", err)
	}

	//_ = rm.defaultRepo.CheckoutRevision("v0.1.2")
	//_ = rm.defaultRepo.CheckoutRevision("develop")
	//_ = rm.defaultRepo.CheckoutRevision("87b65a2d89ce8155ccbc1bd593016f5ff4a3e3d7")
	//_ = rm.defaultRepo.CheckoutRevision("87b65a")
	//_ = rm.RefreshRepos()

	return &rm
}

func (manager *RepoManager) AddRepo(repoPath string) error { //TODO: Improve error handling?
	mutex.Lock()
	defer mutex.Unlock()

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

		_, err = git.PlainClone(repo.GetCloneDir(), false, &git.CloneOptions{
			Auth:   auth,
			URL:    repo.GetUrl(),
			ReferenceName: plumbing.NewBranchReferenceName(repo.Revision),
		})


		if err != nil {
			if err.Error() == "repository already exists" { //Make sure master is checked out
				checkoutErr := repo.CheckoutRevision(repo.Revision)
				if checkoutErr != nil {
					return errors.New(err.Error() + " " + checkoutErr.Error())
				}
			} else {
				cleanErr := cleanCloneParentDirs(repo.GetCloneParentDirs())
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
	mutex.Lock()
	defer mutex.Unlock()

	return manager.repoList
}

func (manager *RepoManager) RemoveRepo(repoPath string) (ok bool) {
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

func (manager *RepoManager) RefreshRepos() error {
	mutex.Lock()
	defer mutex.Unlock()

	for _, repo := range manager.repoList {

		err := repo.RefreshRepo()
		if err != nil {
			return errors.New("Refresh repo for " + repo.GetIdentifier() + ":" + err.Error())
		}
	}

	return nil
}

func (manager *RepoManager) RefreshRepo(repoPath string) error {
	mutex.Lock() //TODO: How does this work???
	defer mutex.Unlock()

	if !strings.HasSuffix(repoPath, "/") { //Add trailing '/'
		repoPath += "/"
	}

	repo := manager.repoList[repoPath]

	return repo.RefreshRepo()
}

func (manager *RepoManager) GetWorkflow(workflowPath string)  (resolvedWorkflowPath string, workflowRepo *Repo, err error) {
	mutex.Lock()
	defer mutex.Unlock()

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
	err = workflowRepo.CheckoutRevision(revision)
	if err != nil { //TODO: This error message doesn't reach coconut
		return
	}

	if !strings.HasSuffix(workflowFile, ".yaml") { //Add trailing ".yaml"
		workflowFile += ".yaml"
	}
	resolvedWorkflowPath = workflowRepo.GetWorkflowDir() + workflowFile

	return
}

func (manager *RepoManager) setDefaultRepo(repo *Repo) {
	if manager.defaultRepo != nil {
		manager.defaultRepo.Default = false //Update old default repo
	}
	manager.defaultRepo = repo
	repo.Default = true
}

func (manager *RepoManager) UpdateDefaultRepo(repoPath string) error {
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
