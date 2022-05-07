package cmd

import (
	"errors"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
	"log"
	"os"

	"squasher/squash"
)

var hours float64
var base string

func RunSquash(path string) {
	context, openErr := squash.OpenRepository(path, base)
	if openErr != nil {
		log.Fatal(openErr)
		return
	}

	log.Printf("Path: %s", path)
	log.Printf("Head: %s (%s)", context.Head.Name(), context.Head.Hash())
	log.Printf("Base: %s", context.BaseCommit)

	if context.Head.Hash() == context.BaseCommit {
		log.Printf("Nothing is changed")
		return
	}

	result, err := squash.Squash(context, hours)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Printf("New commit: %s", result.NewCommit)
	log.Printf("Reduced the number of commits from %d to %d",
		result.NumCommits, result.NewNumCommits,
	)

	newRef := plumbing.NewHashReference(context.Head.Name(), result.NewCommit)
	context.Repository.Storer.SetReference(newRef)

	log.Printf("Updated reference %s", context.Head.Name())
}

var rootCmd = &cobra.Command{
	Use:   "squasher",
	Short: "Squash commits based on commit messages",
	Long:  `FIXME`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("At most one argument is allowed")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) > 0 {
			path = args[0]
		}
		RunSquash(path)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().Float64Var(&hours, "hours", 3, "Maximum time span commits are squashed together")
	rootCmd.Flags().StringVar(&base, "base", "", "Hash of a base commit")
}
