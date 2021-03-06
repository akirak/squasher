package gitutils

import (
	"errors"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var logEnd = errors.New("End of the scanning")

func IsRepoClean(r *git.Repository) (bool, error) {
	worktree, err := r.Worktree()
	if err != nil {
		return false, err
	}

	status, statusErr := worktree.Status()
	if statusErr != nil {
		return false, statusErr
	}

	// At present, IsClean() sometimes returns False, even if the repository
	// is clean. This may be due to a cause described in the following
	// ticket: https://github.com/go-git/go-git/issues/500
	return status.IsClean(), nil
}

func reverse(orig []object.Commit) []object.Commit {
	result := make([]object.Commit, len(orig))
	for i, j := 0, len(orig)-1; i <= j; i, j = i+1, j-1 {
		result[i], result[j] = orig[j], orig[i]
	}
	return result
}

func GetCommitsBetween(repository *git.Repository,
	start plumbing.Hash, openEnd plumbing.Hash) ([]object.Commit, error) {
	logOptions := git.LogOptions{
		From: start,
	}

	iter, logErr := repository.Log(&logOptions)
	if logErr != nil {
		return nil, logErr
	}
	defer iter.Close()

	commits := make([]object.Commit, 0)

	iterErr := iter.ForEach(func(c *object.Commit) error {
		if c.Hash == openEnd {
			return logEnd
		}
		commits = append(commits, *c)
		return nil
	})

	if iterErr != nil && !errors.Is(iterErr, logEnd) {
		return nil, iterErr
	}

	return reverse(commits), nil
}
