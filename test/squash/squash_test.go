package squash

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"squasher/gitutils"
	"squasher/squash"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
)

func GitLog(dir string) (string, error) {
	cmd := exec.Command("git", "log", "--pretty=format:%ad %s", "--date=format:%X", "--reverse")
	cmd.Dir = dir
	output, err := cmd.Output()
	return string(output), err
}

var n = 0

var zone = time.FixedZone("JST", +32400)
var startTime = time.Date(2022, time.January, 1, 0, 0, 0, 0, zone)

func MakeCommit(storer storage.Storer, message string, when time.Time, parent *object.Commit) (*object.Commit, error) {
	author := object.Signature{
		Name:  "squasher",
		Email: "test@squasher.io",
		When:  when,
	}

	committer := object.Signature{
		Name:  "squasher",
		Email: "test@squasher.io",
		When:  time.Now(),
	}

	var content bytes.Buffer
	_, writeErr := content.WriteString(fmt.Sprintf("%d", n))
	n += 1
	if writeErr != nil {
		return nil, writeErr
	}

	blobObj := &plumbing.MemoryObject{}
	blobObj.SetType(plumbing.BlobObject)
	blobObj.Write(content.Bytes())
	blobObj.Close()

	blobHash, blobErr := storer.SetEncodedObject(blobObj)
	if blobErr != nil {
		return nil, blobErr
	}

	treeEntry := object.TreeEntry{
		Name: "sample",
		Mode: filemode.Regular,
		Hash: blobHash,
	}

	treeObj := &plumbing.MemoryObject{}
	treeObj.SetType(plumbing.CommitObject)

	tree := object.Tree{
		Entries: []object.TreeEntry{treeEntry},
	}

	encodedTree := &plumbing.MemoryObject{}
	treeEncodeErr := tree.Encode(encodedTree)
	if treeEncodeErr != nil {
		return nil, treeEncodeErr
	}

	treeHash, treeErr := storer.SetEncodedObject(encodedTree)
	if treeErr != nil {
		return nil, treeErr
	}

	var parentHashes []plumbing.Hash = nil
	if parent != nil {
		parentHashes = []plumbing.Hash{parent.Hash}
	}
	commit := object.Commit{
		Author:       author,
		Committer:    committer,
		Message:      message,
		TreeHash:     treeHash,
		ParentHashes: parentHashes,
	}

	commitObj := &plumbing.MemoryObject{}
	commitEncodeErr := commit.EncodeWithoutSignature(commitObj)
	if commitEncodeErr != nil {
		return nil, commitEncodeErr
	}

	commitHash, commitErr := storer.SetEncodedObject(commitObj)
	if commitErr != nil {
		return nil, commitErr
	}
	commit.Hash = commitHash

	return &commit, nil
}

func Setup(path string) (*squash.SquashContext, error) {
	repo, initError := git.PlainInit(path, false)

	if initError != nil {
		return nil, fmt.Errorf("Failed to init a Git repository: %v", initError)
	}

	storer := repo.Storer

	// Setup

	c0, e0 := MakeCommit(storer, "test 0", startTime, nil)
	if e0 != nil {
		return nil, e0
	}

	c1, e1 := MakeCommit(storer, "test 1", startTime, c0)
	if e1 != nil {
		return nil, e1
	}

	c2, e2 := MakeCommit(storer, "test 1", startTime.Add(120*60*1000000000), c1)
	if e2 != nil {
		return nil, e2
	}

	c3, e3 := MakeCommit(storer, "test 1", startTime.Add(240*60*1000000000), c2)
	if e3 != nil {
		return nil, e3
	}

	c4, e4 := MakeCommit(storer, "test 1", startTime.Add(360*60*1000000000), c3)
	if e4 != nil {
		return nil, e4
	}

	c5, e5 := MakeCommit(storer, "test 2", startTime.Add(390*60*1000000000), c4)
	if e5 != nil {
		return nil, e5
	}

	c6, e6 := MakeCommit(storer, "test 3", startTime.Add(480*60*1000000000), c5)
	if e6 != nil {
		return nil, e6
	}

	branchRef := plumbing.NewBranchReferenceName("master")
	ref1 := plumbing.NewHashReference(branchRef, c6.Hash)
	storer.SetReference(ref1)
	context := squash.SquashContext{
		Repository: repo,
		Head:       ref1,
		BaseCommit: c0.Hash,
	}

	return &context, nil
}

func reverse(orig []object.Commit) []object.Commit {
	result := make([]object.Commit, len(orig))
	for i, j := 0, len(orig)-1; i <= j; i, j = i+1, j-1 {
		result[i], result[j] = orig[j], orig[i]
	}
	return result
}

func GetCommits(repository *git.Repository,
	start plumbing.Hash, openEnd plumbing.Hash) ([]object.Commit, error) {
	commits, commitsErr := gitutils.GetCommitsBetween(repository, start, openEnd)
	if commitsErr != nil {
		return commits, commitsErr
	}
	return reverse(commits), nil
}

func TestSquash3hours(t *testing.T) {
	fmt.Println("3h")

	tmpDir, tmpDirError := ioutil.TempDir("", "squash-test")
	defer os.RemoveAll(tmpDir)

	if tmpDirError != nil {
		t.Fatalf("Failed to create a temporary directory, %v", tmpDirError)
	}

	context, setupErr := Setup(tmpDir)
	if setupErr != nil {
		t.Fatal(setupErr)
	}

	fmt.Println("Before")
	log1Out, log1Err := GitLog(tmpDir)
	if log1Err != nil {
		t.Fatal(log1Err)
	}
	fmt.Println(log1Out)

	result, runErr := squash.Squash(context, 3)
	if runErr != nil {
		t.Fatal(runErr)
	}
	ref2 := plumbing.NewHashReference(context.Head.Name(), result.NewCommit)
	context.Repository.Storer.SetReference(ref2)

	fmt.Println("After")
	log2Out, log2Err := GitLog(tmpDir)
	if log2Err != nil {
		t.Fatal(log2Err)
	}
	fmt.Println(log2Out)

	commits, commitsErr := GetCommits(context.Repository, result.NewCommit, context.BaseCommit)
	if commitsErr != nil {
		t.Fatal(commitsErr)
	}

	if len(commits) != 4 {
		t.Errorf("len(commits) should be %d, but got %d", 4, len(commits))
	}

	if len(commits) > 0 {
		if commits[0].Message != "test 1" {
			t.Errorf("%s should be \"%s\", but got \"%s\"",
				"commits[0].Message",
				"test 1",
				commits[0].Message)
		}

		if !commits[0].Author.When.Equal(startTime) {
			t.Errorf("%s should be 0h0m0s, but got %s",
				"commits[0].Author.When",
				commits[0].Author.When.Sub(startTime).String(),
			)
		}
	}

	if len(commits) > 1 {
		if commits[1].Message != "test 1" {
			t.Errorf("%s should be \"%s\", but got \"%s\"",
				"commits[1].Message",
				"test 1",
				commits[1].Message)
		}

		d, e := time.ParseDuration("4h0m0s")
		if e != nil {
			t.Error(e)
		}
		if !commits[1].Author.When.Equal(startTime.Add(d)) {
			t.Errorf("%s should be 4h0m0s, but got %s",
				"commits[1].Author.When",
				commits[1].Author.When.Sub(startTime).String(),
			)
		}
	}

	if len(commits) > 2 {
		if commits[2].Message != "test 2" {
			t.Errorf("%s should be \"%s\", but got \"%s\"",
				"commits[2].Message",
				"test 2",
				commits[2].Message)
		}

		d, e := time.ParseDuration("6h30m0s")
		if e != nil {
			t.Error(e)
		}
		if !commits[2].Author.When.Equal(startTime.Add(d)) {
			t.Errorf("%s should be 6h30m0s, but got %s",
				"commits[2].Author.When",
				commits[2].Author.When.Sub(startTime).String(),
			)
		}
	}

	if len(commits) > 3 {
		if commits[3].Message != "test 3" {
			t.Errorf("%s should be \"%s\", but got \"%s\"",
				"commits[3].Message",
				"test 3",
				commits[3].Message)
		}

		d, e := time.ParseDuration("8h0m0s")
		if e != nil {
			t.Error(e)
		}
		if !commits[3].Author.When.Equal(startTime.Add(d)) {
			t.Errorf("%s should be 8h0m0s, but got %s",
				"commits[3].Author.When",
				commits[3].Author.When.Sub(startTime).String(),
			)
		}
	}
}
