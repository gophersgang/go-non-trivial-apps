package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	readmeFile = "Readme.md"
)

func main() {
	topLevel()
}

func topLevel() {
	folders := getGitRepos("src/")
	repos := []repo{}
	for _, f := range folders {
		if isDir(f) {
			repo := newRepoForPath(f)
			repos = append(repos, repo)
		}
	}
	readmeHandler{}.replaceProjects(repos)
	for _, repo := range repos {
		fmt.Println(repo.asString())
	}
}

func newRepoForPath(dirPath string) repo {
	fullSize := DirSizeMB(dirPath)
	gitSize := DirSizeMB(dirPath + "/.git")
	codeSize := fullSize - gitSize
	return repo{
		name:     dirPath,
		fullSize: fullSize,
		gitSize:  gitSize,
		codeSize: codeSize,
	}
}

/*
  REPO logic
*/
type repo struct {
	name     string
	fullSize float64
	gitSize  float64
	codeSize float64
}

func (r repo) asString() string {
	name := strings.Replace(r.name, "src/github.com/", "", -1)
	return fmt.Sprintf("%s: %s MB\n  (%s MB git / %s MB code)",
		name,
		floatAsString(r.fullSize),
		floatAsString(r.gitSize),
		floatAsString(r.codeSize),
	)
}

func (r repo) asMarkdown() string {
	name := strings.Replace(r.name, "src/github.com/", "", -1)
	link := fmt.Sprintf("[%s](%s)", name, "https://github.com/"+name)
	return fmt.Sprintf("- %s: %s MB<br/>  (%s MB git / %s MB code)",
		link,
		floatAsString(r.fullSize),
		floatAsString(r.gitSize),
		floatAsString(r.codeSize),
	)
}

func floatAsString(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

// DirSizeMB returns the MB size of a folder
func DirSizeMB(path string) float64 {
	var dirSize int64

	readSize := func(path string, file os.FileInfo, err error) error {
		if !file.IsDir() {
			dirSize += file.Size()
		}
		return nil
	}

	filepath.Walk(path, readSize)
	sizeMB := float64(dirSize) / 1024.0 / 1024.0
	return sizeMB
}

func getGitRepos(path string) []string {
	repos := []string{}
	r := regexp.MustCompile("/.git$")

	findRepos := func(path string, file os.FileInfo, err error) error {
		if file.IsDir() && file.Name() == ".git" {
			newpath := r.ReplaceAllString(path, "")
			repos = append(repos, newpath)
		}
		return nil
	}

	filepath.Walk(path, findRepos)
	return repos
}

func isDir(filePath string) bool {
	fi, err := os.Stat(filePath)
	check(err)
	if fi.IsDir() {
		return true
	}
	return false
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type reposByFullSize []repo

func (ris reposByFullSize) Len() int           { return len(ris) }
func (ris reposByFullSize) Less(i, j int) bool { return ris[i].fullSize > ris[j].fullSize }
func (ris reposByFullSize) Swap(i, j int)      { ris[i], ris[j] = ris[j], ris[i] }

/*
	readmeHandler logic
	rough translation of https://github.com/mindreframer/techwatcher/blob/master/_sh/logic.rb
*/

type readmeHandler struct{}

func (rh readmeHandler) replaceProjects(repos []repo) {
	pattern := `(?s)<!-- SIZE_LIST -->(.*)<!-- /SIZE_LIST -->`
	regexStart := `<!-- SIZE_LIST -->`
	regexEnd := `<!-- /SIZE_LIST -->`

	sort.Sort(reposByFullSize(repos))
	lines := []string{}
	lines = append(lines, regexStart)
	for _, repo := range repos {
		lines = append(lines, repo.asMarkdown())
	}
	lines = append(lines, regexEnd)

	r := regexp.MustCompile(pattern)

	data, err := ioutil.ReadFile(readmeFile)
	check(err)
	res := r.ReplaceAllString(string(data), strings.Join(lines, "\n"))
	ioutil.WriteFile(readmeFile, []byte(res), 0777)
}
