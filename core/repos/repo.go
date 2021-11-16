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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gobwas/glob"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type iRepo interface {
	IRepo
	getCloneParentDirs() []string
	getUri() string
	getWorkflowDir() string
	checkoutRevision(string) error
	refresh() error
	gatherRevisions(*git.Repository) error
	populateWorkflows(string, bool) error
	getWorkflows(string, []string, bool) (TemplatesByRevision, error)
	getHostingSite() string

	getPath() string
	getRepoName() string
	getRevision() string
	setRevision(string)
	getRevisions(string, []string) ([]string, error)
	//getAndSetDefaultRevisionFromRs() string
	setDefaultRevision(string)
	updateDefaultRevision(string) (string, error)
	setDefault(bool)
}

type IRepo interface {
	GetIdentifier() string
	GetCloneDir() string
	ResolveTaskClassIdentifier(string) string
	ResolveSubworkflowTemplateIdentifier(string) string
	GetProtocol() string
	GetHash() string
	GetRevisions() []string
	GetDefaultRevision() string
	IsDefault() bool
	GetTaskTemplatePath(string) string
}

type Repo struct {
	HostingSite     string
	Path            string
	RepoName        string
	Revision        string
	DefaultRevision string
	Hash            string
	Default         bool
	Revisions       []string
	ReposPath		string
}


func resolveProtocolFromPath(repoPath string) string {
	// https
	// starts with github.com/gitlab.cern.ch
	// i.e. has a '.' and no `:` before the first '/'
	//[^.\/]*\.[^\/:]*\/.*
	httpsRegex := `[^.\/]*\.[^\/:]*\/.*`
	re := regexp.MustCompile(httpsRegex)
	if re.Match([]byte(repoPath)) {
		return "https"
	}

	// ssh
	// starts with "hostname:"
	// i.e. has a single ":" before the path
	//[^:]*:[^:]*
	sshRegex := `[^:]*:[^:]*`
	re = regexp.MustCompile(sshRegex)
	if re.Match([]byte(repoPath)) {
		return "ssh"
	}

	// local
	return "local"
}

func newRepo(repoPath string, defaultRevision string, reposPath string) (iRepo, error) {

	protocol, newRepo, err := NewRepo(repoPath, defaultRevision, reposPath)
	if err != nil {
		return nil, err
	}

	if protocol == "ssh" {
		sshRepo := &sshRepo{
			Repo: newRepo,
		}
		return sshRepo, nil
	} else if protocol == "https" {
		httpsRepo := &httpsRepo {
			Repo: newRepo,
		}
		return httpsRepo, nil
	} else if protocol == "local"{
		localRepo := &localRepo {
			Repo: newRepo,
		}
		return localRepo, nil
	}

	return &newRepo, nil
}

func NewRepo(repoPath string, defaultRevision string, reposPath string) (string, Repo, error) {

	// Repo url resolution uses splitAfter(), so if the repoPath ends with "/", it will be split to an extra "" element
	// messing up the resolution. Trim the potential suffix to mitigate the issue
	repoPath = strings.TrimSuffix(repoPath, "/")

	protocol := resolveProtocolFromPath(repoPath)
	if protocol == "local" {
		defaultRevision = "local"
	}

	var repoUrlSlice []string
	var revision string

	revSlice := strings.Split(repoPath, "@")

	if len(revSlice) == 2 { //revision specified in the repo path
		repoUrlSlice = strings.Split(revSlice[0], "/")
		revision = revSlice[1]
	} else if len(revSlice) == 1 { //no revision specified in the repo path
		repoUrlSlice = strings.Split(revSlice[0], "/")
		revision = defaultRevision
	} else {
		return "", Repo{}, errors.New("repo path resolution failed")
	}

	// Discard the "/tasks*" part of the repoPath if this comes from a taskClass
	tasksClassSlice := strings.Split(revSlice[0], "/tasks")

	if protocol == "https" {
		repoUrlSlice = strings.Split(tasksClassSlice[0], "/")
	} else if protocol == "ssh" {
		sshSlice := strings.SplitAfter(tasksClassSlice[0], ":")
		repoUrlSlice = strings.SplitAfter(sshSlice[1], "/")
		repoUrlSlice = append([]string{sshSlice[0]}, repoUrlSlice...)
	} else { // protocol == "local"
		repoUrlSlice = strings.SplitAfter(tasksClassSlice[0], "/")
		repoUrlSlice = append([]string{""}, repoUrlSlice...)
	}

	if len(repoUrlSlice) < 3 {
		return "", Repo{}, errors.New("repo path resolution failed")
	}

	newRepo := Repo{
		HostingSite:     repoUrlSlice[0],
		Path:            path.Join(repoUrlSlice[1 : len(repoUrlSlice)-1]...),
		RepoName:        strings.TrimSuffix(repoUrlSlice[len(repoUrlSlice)-1], ".git"),
		Revision:        revision,
		DefaultRevision: defaultRevision,
		ReposPath:		 reposPath,
	}

	return protocol, newRepo, nil
}

func (r *Repo) GetIdentifier() string {
	return path.Join(r.HostingSite, r.Path, r.RepoName)
}

func (r *Repo) GetCloneDir() string {
	var cloneDir string
	cloneDir = r.ReposPath
	cloneDir = filepath.Join(cloneDir, r.HostingSite, r.Path, r.RepoName)
	return cloneDir
}

func (r *Repo) getCloneParentDirs() []string {
	cleanDir := r.ReposPath

	cleanDirHostingSite := filepath.Join(cleanDir, r.HostingSite)

	cleanDir = cleanDirHostingSite
	dirs := strings.Split(strings.TrimPrefix(r.Path,"/"), "/")
	var cleanDirs []string

	for _, d := range dirs {
		cleanDir = filepath.Join(cleanDir, d)
		cleanDirs = append([]string{cleanDir}, cleanDirs...)
	}
	cleanDirs = append(cleanDirs, cleanDirHostingSite)

	return cleanDirs
}

func (r *Repo) getUri() string {
	u, _ := url.Parse(r.GetProtocol() + "://")
	u.Path = path.Join(u.Path, r.HostingSite)

	u.Path = path.Join(u.Path,
		r.Path,
		r.RepoName)

	uri := u.String()

	return uri + ".git"
}

func (r *Repo) getWorkflowDir() string {
	return filepath.Join(r.GetCloneDir(), "workflows")
}

func (r *Repo) ResolveTaskClassIdentifier(loadTaskClass string) (taskClassIdentifier string) {
	if !strings.Contains(loadTaskClass, "/") {
		taskClassIdentifier = filepath.Join(r.GetIdentifier(),
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
		expr = filepath.Join(r.GetIdentifier(),
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

	ref, err := git.PlainOpen(r.GetCloneDir())
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

	coCmd := exec.Command("git", "-C", r.GetCloneDir(), "checkout", newHash.String())
	err = coCmd.Run()
	if err != nil {
		//try force in the (unlikely) case that something went wrong
		coCmd = exec.Command("git", "-C", r.GetCloneDir(), "checkout", "-f", newHash.String())
		err = coCmd.Run()
		if err != nil {
			return err
		}
	}

	r.Hash = newHash.String() //Update repo hash
	r.Revision = revision     //Update repo revision
	return nil
}

func (r *Repo) refresh() error {
	ref, err := git.PlainOpen(r.GetCloneDir())
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	// clean the repo before doing anything
	// this removes the untracked JIT-produced tasks and workflows
	clnCmd := exec.Command("git", "-C", r.GetCloneDir(), "clean", "-f")
	err = clnCmd.Run()
	if err != nil {
		return errors.New(err.Error() + ": " + r.GetIdentifier())
	}

	err = ref.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Force:      true,
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.New(err.Error() + ": " + r.GetIdentifier() + " | revision: " + r.Revision)
	}

	// gather revisions on update or if empty
	if err != git.NoErrAlreadyUpToDate || r.Revisions == nil {
		err = r.gatherRevisions(ref)
		if err != nil {
			return err
		}
	}

	// populate workflows on update or if empty
	if err != git.NoErrAlreadyUpToDate || len(templatesCache) == 0 {
		err = r.populateWorkflows(r.GetDefaultRevision(), true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repo) gatherRevisions(ref *git.Repository) error {

	var err error
	if ref == nil {
		ref, err = git.PlainOpen(r.GetCloneDir())
		if err != nil {
			return errors.New(err.Error() + ": " + r.GetIdentifier())
		}
	}

	var revs []string
	refs, err := ref.References()
	if err != nil {
		return err
	}
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		// The HEAD is omitted in a `git show-ref` so we ignore the symbolic
		// references, the HEAD
		if ref.Type() == plumbing.SymbolicReference {
			return nil
		}

		nameSlice := strings.Split(ref.Name().String(), "/")
		if len(nameSlice) >= 2 && nameSlice[len(nameSlice)-2] == "origin" {
			revs = append(revs, nameSlice[len(nameSlice)-1])
		}

		return nil
	})
	if err != nil {
		return err
	}

	r.Revisions = revs
	return nil
}

// cache holding the repo's templates
var templatesCache TemplatesByRevision

// updates templatesCache
// triggered with a repo refresh()
func (r *Repo) populateWorkflows(revisionPattern string, clear bool) error {
	// Initialize or clear(!) our cache
	if clear || len(templatesCache) == 0 {
		templatesCache = make(TemplatesByRevision)
	}

	// Include all types of git references
	var gitRefs []string
	gitRefs = append(gitRefs, refRemotePrefix, refTagPrefix)

	// Include all revisions
	revisionsMatched, err := r.getRevisions(revisionPattern, gitRefs)
	if err != nil {
		return err
	}

	for _, revision := range revisionsMatched {
		// Checkout the revision
		err := r.checkoutRevision(revision)
		if err != nil {
			return err
		}

		// Go through the filesystem to locate available templates
		var files []os.FileInfo
		files, err = ioutil.ReadDir(r.getWorkflowDir())
		if err != nil {
			return err
		}
		for _, file := range files {
			// Only return .yaml files
			if strings.HasSuffix(file.Name(), ".yaml") {
				templateName := strings.TrimSuffix(file.Name(), ".yaml")
				workflowPath := filepath.Join(r.getWorkflowDir(), file.Name())
				isPublic, varSpecMap, err := ParseWorkflowPublicVariableInfo(workflowPath)
				if err != nil {
					return err
				}
				templatesCache[RevisionKey(revision)] = append(templatesCache[RevisionKey(revision)],
					Template{templateName, isPublic, varSpecMap})
			}
		}
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
		if len(templatesCache[RevisionKey(revision)]) == 0 {
			err = r.populateWorkflows(revision, false)
			if err != nil {
				return nil, err
			}
		}
		for _, template := range templatesCache[RevisionKey(revision)] {
			// Check if workflow is public in case not allWorkflows requested
			// and skip it if it isn't
			if allWorkflows || (!allWorkflows && template.Public) {
				templates[RevisionKey(revision)] = append(templates[RevisionKey(revision)], template)
			}
		}
	}
	return templates, nil
}

func (r *Repo) getHostingSite() string {
	return r.HostingSite
}

func (r *Repo) GetProtocol() string {
	return "na"
}

func (r *Repo) getPath() string {
	return r.Path
}

func (r *Repo) getRepoName() string {
	return r.RepoName
}

func (r *Repo) GetHash() string {
	return r.Hash
}

func (r *Repo) getRevision() string {
	return r.Revision
}

func (r *Repo) setRevision(revision string) {
	r.Revision = revision
}

func (r *Repo) getRevisions(revisionPattern string, refPrefixes []string) ([]string, error) {
	var revisions []string

	// get a handle of the repo for go-git
	ref, err := git.PlainOpen(r.GetCloneDir())
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

		return nil
	})

	return revisions, nil
}

func (r *Repo) GetRevisions() []string {
	return r.Revisions
}


func (r *Repo) GetDefaultRevision() string {

	return r.DefaultRevision
}

// setDefaultRevision simply assigns, doesn't check
func (r *Repo) setDefaultRevision(revision string) {
	r.DefaultRevision = revision
}

func (r *Repo) updateDefaultRevision(revision string) (string, error) {
	var refs []string
	refs = append(refs, refRemotePrefix, refTagPrefix)
	revisionsMatched, err := r.getRevisions(revision, refs)
	if err != nil {
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


func (r *Repo) IsDefault() bool {
	return r.Default
}

func (r *Repo) setDefault(def bool) {
	r.Default = def
}

func (r *Repo) GetTaskTemplatePath(taskClassFile string) string {
	return filepath.Join(r.ReposPath, taskClassFile)

}