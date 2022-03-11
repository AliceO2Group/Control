/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
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
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type localRepo struct {
	Repo
}

func (r *localRepo) GetCloneDir() string {
	cloneDir := filepath.Join(r.Path, r.RepoName)
	return cloneDir
}

func (r *localRepo) getUri() string {
	u, _ := url.Parse(r.GetProtocol() + "://")
	u.Path = path.Join(u.Path, r.HostingSite)

	u.Path = path.Join(u.Path,
		r.Path,
		r.RepoName)

	uri := u.String()

	return uri + ".git"
}

func (r *localRepo) getWorkflowDir() string {
	return filepath.Join(r.GetCloneDir(), "workflows")
}

func (r *localRepo) ResolveTaskClassIdentifier(loadTaskClass string) (taskClassIdentifier string) {
	if !strings.Contains(loadTaskClass, "/") {
		taskClassIdentifier = filepath.Join(r.GetIdentifier(),
			"tasks",
			loadTaskClass)
	} else {
		taskClassIdentifier = loadTaskClass
	}

	taskClassIdentifier += "@" + r.GetHash()

	return
}

func (r *localRepo) checkoutRevision(string) error {
	// noop for local repo
	return nil
}

func (r *localRepo) refresh() error {
	// noop for local repo
	return nil
}

func (r *localRepo) GetProtocol() string {
	return "local"
}

const timeout = 10 * time.Second
var timer = time.NewTimer(timeout)
var uuidHash string

// GetHash returns a random UUID so that tasks may never be reused
func (r *localRepo) GetHash() string {

	// Generate a UUID and keep it generated for some time
	// This is to allow task matching while creating an environment
	// If a request comes while the timer is running, reset the timer
	// New generation happens on-demand

	select {
	case <-timer.C:
		// timer is finished generate a new UUID
		nuuid, _ := uuid.NewUUID()
		uuidHash = nuuid.String()
		timer.Reset(timeout)
	default:
		// simply reset timeout, keeping existing UUID
		timer.Reset(timeout)
	}

	return uuidHash
}

// getRevisions should make sure that the revisions always contain only "local"
// the placeholder for the localRepo revision
func (r *localRepo) getRevisions(string, []string) ([]string, error) {
	r.Revisions = []string{"local"}
	return r.Revisions, nil
}

func (r *localRepo) GetTaskTemplatePath(taskClassFile string) string {
	return taskClassFile

}

// getWorkflows should always scan for templates, the concept of a template cache is not applicable here
// moreover, revision should always be "local", which is a placeholder for saying no revision applies
func (r *localRepo) getWorkflows(revisionPattern string, _ []string, _ bool) (TemplatesByRevision, error) {
	if revisionPattern != r.GetDefaultRevision() || revisionPattern != "local" {
		return nil, fmt.Errorf("only 'local' revision accepted for local repos")
	}

	revision := "local"
	templates := make(TemplatesByRevision)

	// Go through the filesystem to locate available templates
	files, err := ioutil.ReadDir(r.getWorkflowDir())
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		// Only return .yaml files
		if strings.HasSuffix(file.Name(), ".yaml") {
			templateName := strings.TrimSuffix(file.Name(), ".yaml")
			workflowPath := filepath.Join(r.getWorkflowDir(), file.Name())
			isPublic, description, varSpecMap, err := ParseWorkflowPublicVariableInfo(workflowPath)
			if err != nil {
				return nil, err
			}
			templates[RevisionKey(revision)] = append(templates[RevisionKey(revision)],
				Template{templateName, description, isPublic, varSpecMap})
		}
	}

	return templates, nil;
}