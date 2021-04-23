/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019 CERN and copyright holders of ALICE O².
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
	"errors"
	"github.com/AliceO2Group/Control/common/utils"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/AliceO2Group/Control/apricot"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gobwas/glob"
)

type Repo struct {
	HostingSite string
	User string
	RepoName string
	Revision string
	DefaultRevision string
	Hash string
	Default bool
}

func NewRepo(repoPath string, defaultRevision string) (*Repo, error) {

	revSlice := strings.Split(repoPath, "@")

	var repoUrlSlice []string
	var revision string

	if len(revSlice) == 2 { //revision specified in the repo path
		repoUrlSlice = strings.Split(revSlice[0], "/")
		revision = revSlice[1]
	} else if len(revSlice) == 1 { //no revision specified in the repo path
		repoUrlSlice = strings.Split(revSlice[0], "/")
		revision = defaultRevision
	} else {
		return &Repo{}, errors.New("Repo path resolution failed")
	}

	if len(repoUrlSlice) < 3 {
		return &Repo{}, errors.New("Repo path resolution failed")
	}

	newRepo := Repo{repoUrlSlice[0], repoUrlSlice[1],
		repoUrlSlice[2], revision, defaultRevision, "", false}

	return &newRepo, nil
}

func (r *Repo) GetIdentifier() string {
	return path.Join(r.HostingSite, r.User, r.RepoName)
}

func (r *Repo) getCloneDir() string {
	rs := &RepoService{Svc: apricot.Instance()}
	cloneDir := rs.GetReposPath()
	cloneDir = filepath.Join(cloneDir, r.HostingSite, r.User, r.RepoName)

	return cloneDir
}

func (r *Repo) getCloneParentDirs() []string {
	rs := &RepoService{Svc: apricot.Instance()}
	cleanDir := rs.GetReposPath()

	cleanDirHostingSite := filepath.Join(cleanDir, r.HostingSite)
	cleanDirUser := filepath.Join(cleanDirHostingSite, r.User)

	ret := make([]string, 2)
	ret[0] = cleanDirUser
	ret[1] = cleanDirHostingSite
	return ret
}

func (r *Repo) getUrl() string {
	return "https://" +
		r.HostingSite + "/" +
		r.User + "/" +
		r.RepoName + ".git"
}

func (r *Repo) getWorkflowDir() string {
	return filepath.Join(r.getCloneDir(), "workflows")
}

func (r *Repo) ResolveTaskClassIdentifier(loadTaskClass string) (taskClassIdentifier string) {
	if !strings.Contains(loadTaskClass, "/") {
		taskClassIdentifier = filepath.Join(r.HostingSite,
			r.User,
			r.RepoName,
			"tasks",
			loadTaskClass)
	} else {
		taskClassIdentifier = loadTaskClass
	}

	taskClassIdentifier += "@" + r.Hash

	return
}

func (r *Repo) ResolveSubworkflowTemplateIdentifier(workflowTemplateExpr string) string {
	expr := workflowTemplateExpr
	if !strings.Contains(expr, "/") {
		expr = filepath.Join(r.HostingSite,
			r.User,
			r.RepoName,
			"workflows",
			expr)
	}

	if !strings.Contains(expr, "@") {
		expr += "@" + r.Hash
	}

	return expr
}

func (r *Repo) checkoutRevision(revision string) error {
	if revision == "" {
		revision = r.Revision
	}

	ref, err := git.PlainOpen(r.getCloneDir())
	if err != nil {
		return err
	}

	w, err := ref.Worktree()
	if err != nil {
		return err
	}

	//Try remotely as a priority (branches) so that we don't check out old, dangling branch refs (e.g. master)
	newHash, err := ref.ResolveRevision(plumbing.Revision("origin/" + revision))
	if err != nil {
	 	//Try locally (tags + hashes)
		newHash, err = ref.ResolveRevision(plumbing.Revision(revision))
		if err != nil {
			return errors.New("checkoutRevision: " + err.Error())
		}
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash:  *newHash,
		Force: true,
	})
	if err != nil {
		return err
	}

	r.Hash = newHash.String() //Update repo hash
	r.Revision = revision //Update repo revision
	return nil
}

func (r *Repo) refresh() error {

	ref, err := git.PlainOpen(r.getCloneDir())
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	err = ref.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force: true,
	})

	if err != nil && err.Error() != "already up-to-date" {
		return errors.New(err.Error() + ": " + r.GetIdentifier() + " | revision: " + r.Revision)
	}

	err = r.checkoutRevision(r.DefaultRevision)
	if err != nil {
		return err
	}

	return nil
}

// Returns a map of revision->[]templates for the repo
func (r *Repo) getWorkflows(revisionPattern string, gitRefs []string, allWorkflows bool) (TemplatesByRevision, error) {
	// Get a list of revisions (branches/tags/hash) that are matched by the revisionPattern; gitRefs filter branches and/or tags
	revisionsMatched, err := r.getRevisions(revisionPattern, gitRefs)
	if err != nil {
		return nil, err
	}

	templates := make(TemplatesByRevision)
	for _, revision := range revisionsMatched {

		// Checkout the revision
		err := r.checkoutRevision(revision)
		if err != nil {
			return nil, err
		}

		// Go through the filesystem to locate available templates
		files, err := ioutil.ReadDir(r.getWorkflowDir())
		if err != nil {
			return templates, err
		}
		for _, file := range files {
			// Only return .yaml files
			if strings.HasSuffix(file.Name(), ".yaml") {
				// Check if workflow is public in case not allWorkflows requested
				// and skip it if it isn't
				if allWorkflows || (!allWorkflows &&
					utils.IsFilePublicWorkflow(filepath.Join(r.getWorkflowDir(), file.Name()))) {
					templates[RevisionKey(revision)] = append(templates[RevisionKey(revision)], Template(strings.TrimSuffix(file.Name(), ".yaml")))
				}
			}
		}
	}
	return templates, nil
}


func (r*Repo) getRevisions(revisionPattern string, refPrefixes []string) ([]string, error) {
	var revisions []string

	// get a handle of the repo for go-git
	ref, err := git.PlainOpen(r.getCloneDir())
	if err != nil {
		return nil, errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	// Check if the revision pattern is actually a hash. If so return a single revision
	hashMaybe := plumbing.NewHash(revisionPattern)
	resolvedHash, err := ref.ResolveRevision(plumbing.Revision(revisionPattern))
	if err == nil && *resolvedHash == hashMaybe {
		revisions := append(revisions, resolvedHash.String())
		return revisions, nil
	}

	// If not get a list of git references and loop through them to try and find a match
	refs, err := ref.References()
	if err != nil {
		return nil, errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	// Function that uses the go-git interface for iterating through the references and will populate the revisions slice
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() == plumbing.SymbolicReference { // go-git docs suggests this to skip HEAD, but HEAD makes it anyway
			return nil
		}

		refString := ref.Name().String()
		g := glob.MustCompile(revisionPattern)

		// Function to check the prefix of a git reference to filter branches and/or tags
		resolveRef := func(refString, prefix string) (string, bool) {
			if strings.HasPrefix(refString, prefix) {
				return strings.Split(refString, prefix)[1], true
			}
			return refString, false
		}

		// Loop through the desirable reference prefixes {/refs/tags, refs/remotes/origin} and look for a match
		for _, refPrefix := range refPrefixes {
			if resolvedRefString, ok := resolveRef(refString, refPrefix); ok {
				// In case of a match check the resolved reference string (stripped of the /refs/* prefix)
				// against the revision pattern provided
				if g.Match(resolvedRefString) {
					revisions = append(revisions, resolvedRefString)
					break
				}
			}
		}

		return  nil
	})

	return revisions, nil
}

func (r*Repo) updateDefaultRevision(revision string) (string, error) {
	var refs []string
	refs = append(refs, refRemotePrefix, refTagPrefix)
	revisionsMatched, err := r.getRevisions(revision, refs)
	if err != nil{
		return "", err
	} else if len(revisionsMatched) == 0 {
		revisionsMatched, err = r.getRevisions("*", refs)
		var availableRevs string
		for _, rev := range revisionsMatched {
			availableRevs += rev + "\n"
		}
		return availableRevs, errors.New("revision not found")
	}

	r.DefaultRevision = revision
	return "", nil
}