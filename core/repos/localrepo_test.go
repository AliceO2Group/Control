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

var localTs = ts{
	Protocol:        "local",
	HostingSite:     "",
	Path:            "/home/user/git",
	RepoName:        "ControlWorkflows",
	DefaultRevision: "local",
	RepoPath:        "/home/user/git/ControlWorkflows",
	RepoPathDotGit:  "/home/user/git/ControlWorkflows.git",
}

func TestNewRepoLocal(t *testing.T) {
	repo, err := NewRepo(localTs.RepoPath, localTs.DefaultRevision)
	if err != nil {
		t.Errorf("Creating a new valid repo shouldn't error out")
		return
	}

	if repo.GetProtocol() != localTs.Protocol {
		t.Errorf("Repo's default protocol was %s instead of %s", repo.GetProtocol(), localTs.Protocol)
		return
	}

	if repo.getHostingSite() != localTs.HostingSite {
		t.Errorf("Repo's hosting site was %s instead of %s", repo.getHostingSite(), localTs.HostingSite)
		return
	}

	if repo.getPath() != localTs.Path {
		t.Errorf("Repo's path was %s instead of %s", repo.getPath(), localTs.Path)
		return
	}

	if repo.getRepoName() != localTs.RepoName {
		t.Errorf("Repo name was %s instead of %s", repo.getRepoName(), localTs.RepoName)
		return
	}

	if repo.GetDefaultRevision() != localTs.DefaultRevision {
		t.Errorf("Repo's default revision was %s instead of %s", repo.getRevision(), localTs.DefaultRevision)
		return
	}

	if repo.getRevision() != repo.GetDefaultRevision() {
		t.Errorf("Repo's revision should fall back to default revision, when latter specified. Got %s instead of %s", repo.getRevision(), repo.GetDefaultRevision())
	}
}

func TestGetIdentifierLocal(t *testing.T) {
	repo, _ := NewRepo(localTs.RepoPath, "")
	expectedIdentifier := localTs.Path + "/" + localTs.RepoName

	if repo.GetIdentifier() != expectedIdentifier {
		t.Errorf("Got identifier %s instead of %s", repo.GetIdentifier(), expectedIdentifier)
	}
}

func TestGetUriLocal(t *testing.T) {
	repo, _ := NewRepo(localTs.RepoPath, "")
	u, _ := url.Parse(localTs.Protocol + "://")
	u.Path = path.Join(u.Path,
		localTs.HostingSite,
		localTs.Path,
		localTs.RepoName)
	expectedUri := u.String() + ".git"

	if repo.getUri() != expectedUri {
		t.Errorf("Got %s URI instead of %s", repo.getUri(), expectedUri)
	}
}