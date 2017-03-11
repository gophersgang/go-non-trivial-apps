package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func check(e error) {
	if e != nil {
		log.Println(e)
		panic(e)
	}
}

func checkMsg(e error, msg string) {
	if e != nil {
		log.Println(msg)
		log.Println(e)
		panic(e)
	}
}

func main() {
	s := newSemaphore(20)
	var wg sync.WaitGroup
	urls := loadUrls()
	for _, url := range urls {
		s.Acquire(1)
		wg.Add(1)
		a := url
		go func() {
			defer wg.Done()
			defer s.Release(1)
			checkRepo(a)
		}()
	}
	wg.Wait()
}

func loadUrls() []string {
	return file2lines("data/urls.txt")
}

func checkRepo(url string) error {
	repo := newRepo(url)
	return repo.Run()
}

/*****************************************************************

Repo logic

******************************************************************/

type repo struct {
	url string
}

func newRepo(url string) *repo {
	return &repo{url: url}
}

// Run is the only public method for repos
func (r *repo) Run() error {
	if r.exists() {
		return r.refresh()
	}
	return r.checkout()
}

// initial git checkout
func (r *repo) checkout() error {
	fmt.Printf("checking out %s\n", r.fullPath())
	cmd := fmt.Sprintf("git clone %s %s", r.url, r.fullPath())
	out, err := exec.Command("sh", "-c", cmd).Output()
	checkMsg(err, "For "+r.url)
	fmt.Println(r.projectName(), "\n---\n", string(out))
	return nil
}

// refresh existing repo
func (r *repo) refresh() error {
	fmt.Printf("refreshing %s\n", r.fullPath())
	cmd := fmt.Sprintf("cd %s; git pull", r.fullPath())
	out, err := exec.Command("sh", "-c", cmd).Output()
	checkMsg(err, r.url)
	fmt.Println(r.projectName(), "\n---\n", string(out))
	return nil
}

// does this project exist?
func (r *repo) exists() bool {
	if _, err := os.Stat(r.fullPath()); err == nil {
		return true
	}
	return false
}

// the name of the resulting folder (unique)
func (r *repo) projectName() string {
	parts := strings.Split(r.url, "/")
	user := parts[len(parts)-2]
	name := parts[len(parts)-1]
	name = strings.Replace(name, ".git", "", -1)
	res := fmt.Sprintf("%s--%s", user, name)
	return res
}

// full path to repo folder
func (r *repo) fullPath() string {
	parts := strings.SplitN(r.url, "github.com/", 2)
	return "src/github.com/" + parts[1]
}

/*****************************************************************

Semaphore provides a semaphore synchronization primitive
(vendored for simplicity)

******************************************************************/

// Semaphore controls access to a finite number of resources.
type Semaphore chan struct{}

// New creates a Semaphore that controls access to `n` resources.
func newSemaphore(n int) Semaphore {
	return Semaphore(make(chan struct{}, n))
}

// Acquire `n` resources.
func (s Semaphore) Acquire(n int) {
	for i := 0; i < n; i++ {
		s <- struct{}{}
	}
}

// Release `n` resources.
func (s Semaphore) Release(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

/*
simple lines reader
*/
func file2lines(filePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if t := scanner.Text(); validURL(t) {
			lines = append(lines, t)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return lines
}

func validURL(l string) bool {
	return !strings.Contains(l, " ") && len(l) != 0
}
