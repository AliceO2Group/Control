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
	"net/url"
	"path"
	"testing"
)

var httpTs = ts{
	Protocol:        "https",
	HostingSite:     "github.com",
	Path:            "AliceO2Group",
	RepoName:        "ControlWorkflows",
	DefaultRevision: "main",
	RepoPath:        "github.com/AliceO2Group/ControlWorkflows",
	RepoPathDotGit:  "github.com/AliceO2Group/ControlWorkflows.git",
	ReposPath:       "/var/lib/o2/aliecs/repos",
}

func TestNewRepoInvalidUrl(t *testing.T) {
	repoPathInvalid := "site.com/path"
	_, _, err := NewRepo(repoPathInvalid, "", httpTs.ReposPath)
	if err == nil {
		t.Errorf("NewRepo() should fail if repo URL (non-file) can't be sliced into at least 3 parts")
	}
}

func TestNewRepoDotGit(t *testing.T) {
	_, repo, err := NewRepo(httpTs.RepoPathDotGit, "", httpTs.ReposPath)
	if err != nil {
		t.Errorf("NewRepo() shouldn't fail with valid inputs")
		return
	}

	if repo.getRepoName() != httpTs.RepoName {
		t.Errorf("Repo's name was %s instead of %s", repo.getRepoName(), httpTs.RepoName)
	}
}

func TestNewRepoWithRevision(t *testing.T) {
	revision := "dummy-revision"
	repoPath := httpTs.RepoPath + "@" + revision

	_, repo, err := NewRepo(repoPath, httpTs.DefaultRevision, httpTs.ReposPath)
	if err != nil {
		t.Errorf("Creating a new valid repo shouldn't error out")
		return
	}

	if repo.getRepoName() != httpTs.RepoName {
		t.Errorf("Repo name was %s instead of %s", repo.getRepoName(), httpTs.RepoName)
		return
	}

	if repo.GetDefaultRevision() != httpTs.DefaultRevision {
		t.Errorf("Repo's default revision was %s instead of %s", repo.getRevision(), httpTs.DefaultRevision)
		return
	}

	if repo.getRevision() != revision {
		t.Errorf("Repo's revision should fall back to default revision, when latter specified. Got %s instead of %s", repo.getRevision(), revision)
	}
}

func TestNewRepoWithInvalidRevision(t *testing.T) {
	revision := "dummy-revision"
	repoPath := httpTs.RepoPath + "@" + revision + "@" + revision

	_, _, err := NewRepo(repoPath, httpTs.DefaultRevision, httpTs.ReposPath)
	if err == nil {
		t.Errorf("Creating a new repo with an invalid revision should error out")
		return
	}
}

func TestNewRepoHttps(t *testing.T) {
	_, repo, err := NewRepo(httpTs.RepoPath, httpTs.DefaultRevision, httpTs.ReposPath)
	if err != nil {
		t.Errorf("Creating a new valid repo shouldn't error out")
		return
	}

	if repo.GetProtocol() != httpTs.Protocol {
		t.Errorf("Repo's default protocol was %s instead of %s", repo.GetProtocol(), httpTs.Protocol)
		return
	}

	if repo.getHostingSite() != httpTs.HostingSite {
		t.Errorf("Repo's hosting site was %s instead of %s", repo.getHostingSite(), httpTs.HostingSite)
		return
	}

	if repo.getPath() != httpTs.Path {
		t.Errorf("Repo's path was %s instead of %s", repo.getPath(), httpTs.Path)
		return
	}

	if repo.getRepoName() != httpTs.RepoName {
		t.Errorf("Repo name was %s instead of %s", repo.getRepoName(), httpTs.RepoName)
		return
	}

	if repo.GetDefaultRevision() != httpTs.DefaultRevision {
		t.Errorf("Repo's default revision was %s instead of %s", repo.getRevision(), httpTs.DefaultRevision)
		return
	}

	if repo.getRevision() != repo.GetDefaultRevision() {
		t.Errorf("Repo's revision should fall back to default revision, when latter specified. Got %s instead of %s", repo.getRevision(), repo.GetDefaultRevision())
	}
}

func TestGetUriHttps(t *testing.T) {
	_, repo, _ := NewRepo(httpTs.RepoPath, "", httpTs.ReposPath)
	expectedUri := "https://" + httpTs.HostingSite + "/" + httpTs.Path + "/" + httpTs.RepoName + ".git"

	if repo.getUri() != expectedUri {
		t.Errorf("Got %s URI instead of %s", repo.getUri(), expectedUri)
	}
}

func TestGetUriHttpsWithDotGit(t *testing.T) {
	_, repo, _ := NewRepo(httpTs.RepoPathDotGit, "", httpTs.ReposPath)
	u, _ := url.Parse(httpTs.Protocol + "://")
	u.Path = path.Join(u.Path,
		httpTs.HostingSite,
		httpTs.Path,
		httpTs.RepoName)
	expectedUri := u.String() + ".git"

	if repo.getUri() != expectedUri {
		t.Errorf("Got %s URI instead of %s", repo.getUri(), expectedUri)
	}

}

func TestGetIdentifierHttps(t *testing.T) {
	_, repo, _ := NewRepo(httpTs.RepoPath, "", httpTs.ReposPath)
	expectedIdentifier := path.Join(httpTs.HostingSite, httpTs.Path, httpTs.RepoName)

	if repo.GetIdentifier() != expectedIdentifier {
		t.Errorf("Got identifier %s instead of %s", repo.GetIdentifier(), expectedIdentifier)
	}
}

func TestGetIdentifierHttpsDotGit(t *testing.T) {
	_, repo, _ := NewRepo(httpTs.RepoPathDotGit, "", httpTs.ReposPath)
	expectedIdentifier := path.Join(httpTs.HostingSite, httpTs.Path, httpTs.RepoName)

	if repo.GetIdentifier() != expectedIdentifier {
		t.Errorf("Got identifier %s instead of %s", repo.GetIdentifier(), expectedIdentifier)
	}
}
