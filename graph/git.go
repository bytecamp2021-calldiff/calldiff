package graph

import (
	"calldiff/common"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Clone a repository using clone options
func clone(url, dir string) *git.Repository {
	// check whether it's necessary to clone git repo
	if _, err := os.Stat(dir); os.IsNotExist(err) && url != "" {
		// Clone the given repository to the given directory
		common.Info("git clone %s %s --recursive", url, dir)

		r, err := git.PlainClone(dir, false, &git.CloneOptions{
			URL:               url,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		})
		common.CheckIfError(err)

		return r
	} else {
		r, err := git.PlainOpen(dir)
		common.CheckIfError(err)

		return r
	}
}

func getCommitHash(r *git.Repository, s string) *object.Commit {
	var hash plumbing.Hash
	head, err := r.Head()
	common.CheckIfError(err)
	if s == "HEAD" {
		hash = head.Hash()
	} else if s == "HEAD^" {
		ref, err := r.CommitObject(head.Hash())
		common.CheckIfError(err)

		commitIter := object.NewCommitIterCTime(ref, nil, nil)

		_, err = commitIter.Next() // at HEAD
		common.CheckIfError(err)

		commit, err := commitIter.Next() // at HEAD^
		common.CheckIfError(err)

		hash = commit.Hash
	} else {
		hash = plumbing.NewHash(s)
	}
	commit, err := r.CommitObject(hash)
	common.CheckIfError(err)
	return commit
}

func outputCommitFiles(commit *object.Commit, dir string) error {
	filesIter, err := commit.Files()
	if err != nil {
		return err
	}
	for {
		file, err := filesIter.Next()
		if err != nil { // err 只可能等于 nil 或 io.EOF
			break
		}
		if file.Name != "go.mod" && file.Name != "go.sum" && !strings.HasSuffix(file.Name, ".go") {
			continue
		}
		contents, err := file.Contents()
		if err != nil {
			return err
		}
		filePath := dir + "/" + file.Name
		if err := os.MkdirAll(filepath.Dir(filePath), fs.ModePerm); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filePath, []byte(contents), fs.ModePerm); err != nil {
			return err
		}
	}
	return nil
}
