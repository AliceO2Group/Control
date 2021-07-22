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

var sshTs = ts{
	Protocol:        "ssh",
	HostingSite:     "git-server:",
	Path:            "/opt/git",
	RepoName:        "ControlWorkflows",
	DefaultRevision: "main",
	RepoPath:        "git-server:/opt/git/ControlWorkflows",
	RepoPathDotGit:  "git-server:/opt/git/ControlWorkflows.git",
}

func TestGetUriSsh(t *testing.T) {
	repo, _ := NewRepo(sshTs.RepoPath, "")
	u, _ := url.Parse(sshTs.Protocol + "://git@" + sshTs.HostingSite)
	u.Path = path.Join(u.Path,
		sshTs.Path,
		sshTs.RepoName)
	expectedUri := u.String() + ".git"

	if repo.getUri() != expectedUri {
		t.Errorf("Got %s URI instead of %s", repo.getUri(), expectedUri)
	}
}

func TestGetIdentifierSsh(t *testing.T) {
	repo, _ := NewRepo(sshTs.RepoPath, "")
	expectedIdentifier := path.Join(sshTs.HostingSite, sshTs.Path, sshTs.RepoName)

	if repo.GetIdentifier() != expectedIdentifier {
		t.Errorf("Got identifier %s instead of %s", repo.GetIdentifier(), expectedIdentifier)
	}
}

func TestNewRepoSsh(t *testing.T) {
	repo, err := NewRepo(sshTs.RepoPath, sshTs.DefaultRevision)
	if err != nil {
		t.Errorf("Creating a new valid repo shouldn't error out")
		return
	}

	if repo.GetProtocol() != sshTs.Protocol {
		t.Errorf("Repo's default protocol was %s instead of %s", repo.GetProtocol(), sshTs.Protocol)
		return
	}

	if repo.getHostingSite() != sshTs.HostingSite {
		t.Errorf("Repo's hosting site was %s instead of %s", repo.getHostingSite(), sshTs.HostingSite)
		return
	}

	if repo.getPath() != sshTs.Path {
		t.Errorf("Repo's path was %s instead of %s", repo.getPath(), sshTs.Path)
		return
	}

	if repo.getRepoName() != sshTs.RepoName {
		t.Errorf("Repo name was %s instead of %s", repo.getRepoName(), sshTs.RepoName)
		return
	}

	if repo.GetDefaultRevision() != sshTs.DefaultRevision {
		t.Errorf("Repo's default revision was %s instead of %s", repo.getRevision(), sshTs.DefaultRevision)
		return
	}

	if repo.getRevision() != repo.GetDefaultRevision() {
		t.Errorf("Repo's revision should fall back to default revision, when latter specified. Got %s instead of %s", repo.getRevision(), repo.GetDefaultRevision())
	}
}