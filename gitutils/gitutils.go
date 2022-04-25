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

	return status.IsClean(), nil
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

	if iterErr != nil && ! errors.Is(iterErr, logEnd) {
		return nil, iterErr
	}

	return commits, nil
}
