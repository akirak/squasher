package squash

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"squasher/gitutils"
)

type SquashContext struct {
	Repository *git.Repository
	Head       *plumbing.Reference
	BaseCommit plumbing.Hash
}

func OpenRepository(path string) (*SquashContext, error) {
	repo, openErr := git.PlainOpen(path)
	if openErr != nil {
		return nil, openErr
	}

	clean, cleanErr := gitutils.IsRepoClean(repo)
	if cleanErr != nil {
		return nil, cleanErr
	}
	if !clean {
		return nil, fmt.Errorf("Repository is not clean")
	}

	head, headErr := repo.Head()
	if headErr != nil {
		return nil, headErr
	}
	headName := head.Name()
	if !headName.IsBranch() {
		return nil, fmt.Errorf("%s is not a branch", headName)
	}
	branch := headName.Short()

	config, configErr := repo.Config()
	if configErr != nil {
		return nil, configErr
	}

	remote := ""
	branchConfig, branchConfigExists := config.Branches[branch]
	if branchConfigExists {
		remote = branchConfig.Remote
	} else {
		// go-git seems to have no API for retrieving
		// remote.pushDefault, so use "origin" as the push default.
		remote = "origin"
	}

	remoteRef, remoteErr := repo.Reference(
		plumbing.NewRemoteReferenceName(remote, branch), false)
	if remoteErr != nil {
		return nil, remoteErr
	}
	parentHash := remoteRef.Hash()

	context := SquashContext{
		Repository: repo,
		Head:       head,
		BaseCommit: parentHash,
	}

	return &context, nil
}

type SquashResult struct {
	NewCommit     plumbing.Hash
	NumCommits    int
	NewNumCommits int
}

func Squash(context *SquashContext, thresholdInHours float64) (*SquashResult, error) {
	commits, logErr := gitutils.GetCommitsBetween(context.Repository,
		context.Head.Hash(),
		context.BaseCommit)
	if logErr != nil {
		return nil, logErr
	}

	currentCommit := commits[len(commits)-1]
	time := currentCommit.Author.When
	message := currentCommit.Message

	num := 0
	parentHash := context.BaseCommit
	storer := context.Repository.Storer

	for j := len(commits) - 1; j >= 0; j-- {
		newMessage := commits[j].Message
		newTime := commits[j].Author.When
		duration := newTime.Sub(time)

		isSquash := message == newMessage && duration.Hours() < thresholdInHours

		newAuthor := currentCommit.Author
		if !isSquash {
			newAuthor = commits[j].Author
		}

		parentHashes := currentCommit.ParentHashes
		if !isSquash {
			parentHashes = []plumbing.Hash{parentHash}
		}

		committer := object.Signature{
			Name:  newAuthor.Name,
			Email: newAuthor.Email,
			When:  time.Local(),
		}

		newCommit := object.Commit{
			Author:       newAuthor,
			Committer:    committer,
			Message:      commits[j].Message,
			TreeHash:     commits[j].TreeHash,
			ParentHashes: parentHashes,
		}

		encoded := &plumbing.MemoryObject{}
		encodeErr := newCommit.Encode(encoded)
		if encodeErr != nil {
			return nil, encodeErr
		}

		newHash, saveErr := storer.SetEncodedObject(encoded)
		if saveErr != nil {
			return nil, saveErr
		}
		newCommit.Hash = newHash

		currentCommit = newCommit

		if !isSquash {
			time = commits[j].Author.When
			message = newMessage
			parentHash = newHash
			num += 1
		}
	}

	result := SquashResult{
		NewCommit:     currentCommit.Hash,
		NumCommits:    len(commits),
		NewNumCommits: num,
	}

	return &result, nil
}
