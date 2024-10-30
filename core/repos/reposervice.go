package repos

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/AliceO2Group/Control/configuration"
	"github.com/spf13/viper"
)

// TODO: remove support for FILE backend in this one
type RepoService struct {
	Svc configuration.RuntimeService
}

func (s *RepoService) GetReposPath() string {
	return filepath.Join(viper.GetString("coreWorkingDir"), "repos")
}

func (s *RepoService) NewDefaultRepo(defaultRepo string) error {
	if s.Svc != nil {
		return s.Svc.SetRuntimeEntry("aliecs", "default_repo", defaultRepo)
	} else {
		data := []byte(defaultRepo)
		return ioutil.WriteFile(filepath.Join(s.GetReposPath(), "default_repo"), data, 0644)
	}
}

func (s *RepoService) GetDefaultRepo() (defaultRepo string, err error) {
	if s.Svc != nil {
		return s.Svc.GetRuntimeEntry("aliecs", "default_repo")
	} else {
		var defaultRepoData []byte
		defaultRepoData, err = ioutil.ReadFile(filepath.Join(s.GetReposPath(), "default_repo"))
		if err != nil {
			return
		}
		defaultRepo = strings.TrimSuffix(string(defaultRepoData), "\n")
		return
	}
}

func (s *RepoService) NewDefaultRevision(defaultRevision string) error {
	if s.Svc != nil {
		return s.Svc.SetRuntimeEntry("aliecs", "default_revision", defaultRevision)
	} else {
		data := []byte(defaultRevision)
		return ioutil.WriteFile(filepath.Join(s.GetReposPath(), "default_revision"), data, 0644)
	}
}

func (s *RepoService) GetDefaultRevision() (defaultRevision string, err error) {
	if s.Svc != nil {
		return s.Svc.GetRuntimeEntry("aliecs", "default_revision")
	} else {
		var defaultRevisionData []byte
		defaultRevisionData, err = ioutil.ReadFile(filepath.Join(s.GetReposPath(), "default_revision"))
		if err != nil {
			return
		}
		defaultRevision = strings.TrimSuffix(string(defaultRevisionData), "\n")
		return
	}
}

func (s *RepoService) GetRepoDefaultRevisions() (map[string]string, error) {
	var defaultRevisions map[string]string
	if s.Svc != nil {
		data, err := s.Svc.GetRuntimeEntry("aliecs", "default_revisions")
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(data), &defaultRevisions)
		if err != nil {
			return nil, err
		}
	} else {
		defaultRevisionData, err := ioutil.ReadFile(filepath.Join(s.GetReposPath(), "default_revisions.json"))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(defaultRevisionData, &defaultRevisions)
	}
	return defaultRevisions, nil
}

func (s *RepoService) SetRepoDefaultRevisions(defaultRevisions map[string]string) error {
	data, err := json.MarshalIndent(defaultRevisions, "", "    ")
	if err != nil {
		return err
	}

	if s.Svc != nil {
		err = s.Svc.SetRuntimeEntry("aliecs", "default_revisions", string(data))
	} else {
		err = ioutil.WriteFile(filepath.Join(s.GetReposPath(), "default_revisions.json"), data, 0644)
	}
	return err
}
