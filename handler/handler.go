package handler

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	github "github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var client *github.Client
var ctx = context.Background()
var data Data

type Data struct {
	sourceOwner   *string
	sourceRepo    *string
	commitMessage *string
	commitBranch  *string
	baseBranch    *string
	prRepo        *string
	prBranch      *string
	authorName    *string
	authorEmail   *string
	sourceFiles   *string
	prRepoOwner   *string
	prSubject     *string
	prDescription *string
}

// /login
func HandleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	Init()
	tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: "120157184fe25c6efc2033138a474b56fafd6503"},
	)
	tokenClient := oauth2.NewClient(ctx, tokenService)

	client = github.NewClient(tokenClient)

	// create a new private repository
	newRepo := &github.Repository{
		Name:    github.String("newRepo"),
		Private: github.Bool(false),
	}
	client.Repositories.Create(ctx, "", newRepo)
	fmt.Println("NewRepo got created")
	fmt.Fprintf(w, "<h1>Created new repo. </h1>")
	fileContent := []byte("This is the content of my file\nand the 2nd line of it")

	// Note: the file needs to be absent from the repository as you are not
	// specifying a SHA reference here.
	opts := &github.RepositoryContentFileOptions{
		Message:   github.String("This is my commit message"),
		Content:   fileContent,
		Branch:    github.String("master"),
		Committer: &github.CommitAuthor{Name: github.String("FirstName LastName"), Email: github.String("user@example.com")},
	}
	_, _, err := client.Repositories.CreateFile(ctx, "sh4ggyy", "newRepo", "myNewFile.md", opts)
	if err != nil {
		fmt.Println(err)
		return
	}

	ref, err := getRef()
	if err != nil {
		log.Fatalf("Unable to get/create the commit reference: %s\n", err)
	}
	if ref == nil {
		log.Fatalf("No error where returned but the reference is nil")
	}

	tree, err := getTree(ref)
	if err != nil {
		log.Fatalf("Unable to create the tree based on the provided files: %s\n", err)
	}

	if err := pushCommit(ref, tree); err != nil {
		log.Fatalf("Unable to create the commit: %s\n", err)
	}

	if err := createPR(); err != nil {
		log.Fatalf("Error while creating the pull request: %s", err)
	}
}

func Init() {
	SourceOwner := "sh4ggyy"
	SourceRepo := "newRepo"
	CommitMessage := "commit-message"
	CommitBranch := "commit-branch"
	BaseBranch := "master"
	PrRepo := ""
	PrBranch := "master"
	AuthorName := "sh4ggyy"
	AuthorEmail := "gmail"
	SourceFiles := "file.txt"
	PrRepoOwner := "sh4ggyy"
	PrSubject := "PR"
	PrDescription := "desc"

	data = Data{
		sourceOwner:   &SourceOwner,
		sourceRepo:    &SourceRepo,
		commitMessage: &CommitMessage,
		commitBranch:  &CommitBranch,
		baseBranch:    &BaseBranch,
		prRepo:        &PrRepo,
		prBranch:      &PrBranch,
		authorName:    &AuthorName,
		authorEmail:   &AuthorEmail,
		sourceFiles:   &SourceFiles,
		prRepoOwner:   &PrRepoOwner,
		prSubject:     &PrSubject,
		prDescription: &PrDescription,
	}
}

// getRef returns the commit branch reference object if it exists or creates it
// from the base branch before returning it.
func getRef() (ref *github.Reference, err error) {
	if ref, _, err = client.Git.GetRef(ctx, *data.sourceOwner, *data.sourceRepo, "refs/heads/"+*data.commitBranch); err == nil {
		return ref, nil
	}

	// We consider that an error means the branch has not been found and needs to
	// be created.
	if *data.commitBranch == *data.baseBranch {
		return nil, errors.New("The commit branch does not exist but `-base-branch` is the same as `-commit-branch`")
	}

	if *data.baseBranch == "" {
		return nil, errors.New("The `-base-branch` should not be set to an empty string when the branch specified by `-commit-branch` does not exists")
	}

	var baseRef *github.Reference
	if baseRef, _, err = client.Git.GetRef(ctx, *data.sourceOwner, *data.sourceRepo, "refs/heads/"+*data.baseBranch); err != nil {
		return nil, err
	}
	newRef := &github.Reference{Ref: github.String("refs/heads/" + *data.commitBranch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = client.Git.CreateRef(ctx, *data.sourceOwner, *data.sourceRepo, newRef)
	return ref, err
}

func getTree(ref *github.Reference) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}

	// Load each file into the tree.
	for _, fileArg := range strings.Split(*data.sourceFiles, ",") {
		file, content, err := getFileContent(fileArg)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &github.TreeEntry{Path: github.String(file), Type: github.String("blob"), Content: github.String(string(content)), Mode: github.String("100644")})
	}

	tree, _, err = client.Git.CreateTree(ctx, *data.sourceOwner, *data.sourceRepo, *ref.Object.SHA, entries)
	return tree, err
}

// getFileContent loads the local content of a file and return the target name
// of the file in the target repository and its contents.
func getFileContent(fileArg string) (targetName string, b []byte, err error) {
	var localFile string
	files := strings.Split(fileArg, ":")
	switch {
	case len(files) < 1:
		return "", nil, errors.New("empty `-files` parameter")
	case len(files) == 1:
		localFile = files[0]
		targetName = files[0]
	default:
		localFile = files[0]
		targetName = files[1]
	}

	b, err = ioutil.ReadFile(localFile)
	return targetName, b, err
}

// pushCommit creates the commit in the given reference using the given tree.
func pushCommit(ref *github.Reference, tree *github.Tree) (err error) {
	// Get the parent commit to attach the commit to.
	parent, _, err := client.Repositories.GetCommit(ctx, *data.sourceOwner, *data.sourceRepo, *ref.Object.SHA)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	authorEmail := data.authorEmail
	author := &github.CommitAuthor{Date: &date, Name: data.authorName, Email: authorEmail}
	commit := &github.Commit{Author: author, Message: data.commitMessage, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	newCommit, _, err := client.Git.CreateCommit(ctx, *data.sourceOwner, *data.sourceRepo, commit)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = client.Git.UpdateRef(ctx, *data.sourceOwner, *data.sourceRepo, ref, false)
	return err
}

func createPR() (err error) {
	if *data.prSubject == "" {
		return errors.New("missing `-pr-title` flag; skipping PR creation")
	}

	if *data.prRepoOwner != "" && *data.prRepoOwner != *data.sourceOwner {
		*data.commitBranch = fmt.Sprintf("%s:%s", *data.sourceOwner, *data.commitBranch)
	} else {
		*data.prRepoOwner = *data.sourceOwner
	}

	if *data.prRepo == "" {
		*data.prRepo = *data.sourceRepo
	}

	newPR := &github.NewPullRequest{
		Title:               data.prSubject,
		Head:                data.commitBranch,
		Base:                data.prBranch,
		Body:                data.prDescription,
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := client.PullRequests.Create(ctx, *data.prRepoOwner, *data.prRepo, newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}
