package task

import (
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	once sync.Once
	instance *Repos
	mutex sync.Mutex
)

func BindToInstance(m *Manager) *Repos {
	once.Do(func() {
		instance = initializeRepos(m)
	})
	return instance
}

func ReposInstance() *Repos {
	return instance
}

type Repos struct {
	repoList map[string]bool //I want this to work as a set
	taskMan *Manager
}

func initializeRepos(m *Manager) *Repos {
	log.Debug("Initializing repos singleton")
	return &Repos{repoList: map[string]bool {viper.GetString("defaultRepo"):true}, taskMan: m}
}

func (repos *Repos) populateRepoList() (err error) {
	//TODO: Go through the dir and get whatever git info you can
	return
}

func (repos *Repos) AddRepo(repoPath string) (bool) {
	mutex.Lock()
	defer mutex.Unlock()

	_, exists := repos.repoList[repoPath]
	if !exists {
		repos.repoList[repoPath] = true
		return true
	} else {
		return false
	}
}

func (repos *Repos) GetRepos() (repoList map[string]bool, err error) {
	return repos.repoList, nil
}

func (repos *Repos) RefreshRepos() (err error) { //TODO: One, more or all?
	//TODO: Fill me
	return
}

func (repos *Repos) RemoveRepo(repoPath string) (err error) {
	mutex.Lock()
	defer mutex.Unlock()

	_, exists := repos.repoList[repoPath]
	if exists {
		delete(repos.repoList, repoPath)
	}
	return
}

func (repos *Repos) RefreshClasses() (taskClassesList []*TaskClass, err error) {
	mutex.Lock()
	defer mutex.Unlock()

	var yamlData []byte

	taskClassesList = make([]*TaskClass, 0)

	repoList, _ := repos.GetRepos()

	for repo := range repoList {
		var taskFiles []os.FileInfo
		taskFilesDir := viper.GetString("repositoriesUri") + repo + "tasks/"
		taskFiles, err = ioutil.ReadDir(taskFilesDir);
		for _, file := range taskFiles {
			if filepath.Ext(file.Name()) != ".yaml" {
				continue
			}
			yamlData, err = ioutil.ReadFile(taskFilesDir + file.Name())
			if err != nil {
				return nil, err
			}
			//var taskClass *TaskClass //TODO: This doesn't unmarshal; unclear why
			//taskClass = new(TaskClass)
			taskClass := make([]*TaskClass, 0)
			err = yaml.Unmarshal(yamlData, &taskClass)
			if err != nil {
				return nil, err
			}

			taskClass[0].Repo = repo
			taskClassesList = append(taskClassesList, taskClass ...)
		}
	}
	return taskClassesList, nil
}

func (repos *Repos) GetWorkflow(workflowPath string)  (resolvedWorkflowPath string, workflowRepo string, err error) {
	workflowInfo := strings.Split(workflowPath, "workflows/")
	workflowRepo = workflowInfo[0]
	if workflowRepo == "" {
		workflowRepo = viper.GetString("defaultRepo")
	}
	//workflowDir := workflowInfo[1]

	// Refresh task classes if repo was not already included
	added := repos.AddRepo(workflowRepo)
	if added { //Only refresh classes if repo was newly added
		_ = repos.taskMan.RefreshClasses()
	}

	// Resolve actual workflow path
	if strings.Contains(workflowPath, "/") {
		resolvedWorkflowPath = viper.GetString("repositoriesUri") +  workflowPath + ".yaml"
	} else {
		resolvedWorkflowPath = viper.GetString("repositoriesUri") + viper.GetString("defaultRepo")  +
		"workflows/" + workflowPath + ".yaml"
	}

	return resolvedWorkflowPath, workflowRepo, nil
	//TODO: error handling
}

type RepoProperties struct {
	propA string
	propB string
}

func (repos Repos) SetRepoProperties(repoPath string, properties RepoProperties) (err error) {
	//TODO: Fill me
	return
}

func TraverseTasksPath() (err error) {
	return
}

func TraverseWorkflowsPath() (err error) {
	return
}