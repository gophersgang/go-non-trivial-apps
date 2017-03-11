package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var (
	urlFile    = "data/urls.txt"
	readmeFile = "Readme.md"
)

func main() {
	var wg sync.WaitGroup
	urls := loadUrls()
	repos := []repoInfo{}
	for _, url := range urls {
		wg.Add(1)
		a := url
		go func() {
			defer wg.Done()
			repo := process(a)
			repos = append(repos, repo)
			fmt.Print(".")
		}()
	}
	wg.Wait()
	printSortedAlpha(repos)
	printSortedLastcommit(repos)
	rh := readmeHandler{}
	rh.replaceProjects(repos)
	rh.replaceActivity(repos)
}

func printSortedAlpha(repos []repoInfo) {
	sort.Sort(reposByURL(repos))
	fmt.Print("\n\n")
	for _, r := range repos {
		fmt.Println(r.MarkdownProject())
	}
}

func printSortedLastcommit(repos []repoInfo) {
	sort.Sort(reposByLastcommit(repos))
	fmt.Print("\n\n")
	for _, r := range repos {
		fmt.Println(r.MarkdownActivity())
	}
}

func process(url string) repoInfo {
	parser := repoParser{}
	doc := parser.getDoc(url)
	repo := repoInfo{
		url:          strings.ToLower(url),
		description:  parser.getDescription(doc),
		lastcommit:   parser.getLastcommit(doc),
		commitsCount: parser.getCommitsCount(doc),
		stars:        parser.getStarsCount(doc),
	}
	return repo
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func checkMsg(e error, msg string) {
	if e != nil {
		log.Println(msg)
		log.Println(e)
		panic(e)
	}
}

/*
	readmeHandler logic
	rough translation of https://github.com/mindreframer/techwatcher/blob/master/_sh/logic.rb
*/

type readmeHandler struct{}

func (rh readmeHandler) replaceProjects(repos []repoInfo) {
	pattern := `(?s)<!-- PROJECTS_LIST -->(.*)<!-- /PROJECTS_LIST -->`
	regexStart := `<!-- PROJECTS_LIST -->`
	regexEnd := `<!-- /PROJECTS_LIST -->`

	sort.Sort(reposByURL(repos))
	lines := []string{}
	lines = append(lines, regexStart)
	for _, repo := range repos {
		lines = append(lines, repo.MarkdownProject())
	}
	lines = append(lines, regexEnd)

	r := regexp.MustCompile(pattern)

	data, err := ioutil.ReadFile(readmeFile)
	checkMsg(err, readmeFile)
	res := r.ReplaceAllString(string(data), strings.Join(lines, "\n"))
	ioutil.WriteFile(readmeFile, []byte(res), 0777)
}

func (rh readmeHandler) replaceActivity(repos []repoInfo) {
	pattern := `(?s)<!-- ACTIVITY_LIST -->(.*)<!-- /ACTIVITY_LIST -->`
	regexStart := `<!-- ACTIVITY_LIST -->`
	regexEnd := `<!-- /ACTIVITY_LIST -->`

	sort.Sort(reposByLastcommit(repos))
	lines := []string{}
	lines = append(lines, regexStart)
	for _, repo := range repos {
		lines = append(lines, repo.MarkdownActivity())
	}
	lines = append(lines, regexEnd)
	r := regexp.MustCompile(pattern)

	data, err := ioutil.ReadFile(readmeFile)
	check(err)
	res := r.ReplaceAllString(string(data), strings.Join(lines, "\n"))
	ioutil.WriteFile(readmeFile, []byte(res), 0777)
}

/*
	Repo Parser logic
*/
type repoParser struct{}

func (r repoParser) getDescription(doc *goquery.Document) string {
	var content string
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		if name == "description" {
			content, _ = s.Attr("content")
		}
	})
	var parts []string
	if strings.Contains(content, " - ") {
		parts = strings.SplitN(content, " - ", 2)
		content = parts[1]
	}
	if strings.Contains(content, "by creating an account on GitHub") {
		content = "---"
	}

	return strings.TrimSpace(content)
}

func (r repoParser) getCommitsCount(doc *goquery.Document) string {
	return r.getSelectorText(doc, ".numbers-summary .commits .num")
}

func (r repoParser) getStarsCount(doc *goquery.Document) string {
	return r.getSelectorText(doc, ".social-count")
}

func (r repoParser) getLastcommit(doc *goquery.Document) string {
	if r.hasIncludedLastcommit(doc) {
		return r.getLastcommitIncluded(doc)
	}
	return r.getLastcommitAjax(doc)
}

func (r repoParser) hasIncludedLastcommit(doc *goquery.Document) bool {
	found := true
	doc.Find(".commit-loader").Each(func(i int, s *goquery.Selection) {
		found = false
	})
	return found
}

func (r repoParser) getLastcommitIncluded(doc *goquery.Document) string {
	var datetime string
	doc.Find(".commit-tease relative-time").Each(func(i int, s *goquery.Selection) {
		datetime, _ = s.Attr("datetime")
	})
	return datetime
}

func (r repoParser) getLastcommitAjax(doc *goquery.Document) string {
	// extract the ajax url
	// e.g.: <include-fragment class="commit-tease commit-loader" src="/f2prateek/coi/tree-commit/866dee22e2b11dd9780770c00bae53886d9b4863">
	s := doc.Find(".commit-loader")
	path, _ := s.Attr("src")
	url := "https://github.com" + path
	ajaxDoc := r.urlDoc(url)
	return r.getLastcommit(ajaxDoc)
}

func (r repoParser) getSelectorText(doc *goquery.Document, selector string) string {
	var content string
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		content = strings.TrimSpace(s.Text())
	})
	return content
}

func (r repoParser) getDoc(url string) *goquery.Document {
	return r.urlDoc(url)
	// return r.localDoc()
}

func (r repoParser) urlDoc(url string) *goquery.Document {
	doc, err := goquery.NewDocument(url)
	checkMsg(err, url)
	return doc
}

func (r repoParser) localDoc() *goquery.Document {
	filename := "crawler/fixture.html"
	file, err := os.Open(filename)
	check(err)
	doc, err := goquery.NewDocumentFromReader(file)
	check(err)
	return doc
}

/*
	repoInfo logic
*/
type repoInfo struct {
	url          string
	description  string
	lastcommit   string
	commitsCount string
	stars        string
}

func (ri repoInfo) MarkdownProject() string {
	return fmt.Sprintf("- %s - %s <br/> ( %s / %s commits / %s stars )",
		ri.mdLink(),
		ri.description,
		ri.lastcommitShort(),
		ri.commitsCount,
		ri.stars,
	)
}

func (ri repoInfo) MarkdownActivity() string {
	return fmt.Sprintf("- %s: %s <br/> %s",
		ri.mdLink(),
		ri.lastcommitShort(),
		ri.description,
	)
}

func (ri repoInfo) shorturl() string {
	return strings.Replace(ri.url, "https://github.com/", "", -1)
}

func (ri repoInfo) mdLink() string {
	return fmt.Sprintf("[%s](%s)", ri.shorturl(), ri.url)
}

func (ri repoInfo) lastcommitShort() string {
	return ri.lastcommit[0:10]
}

type reposByLastcommit []repoInfo

func (ris reposByLastcommit) Len() int           { return len(ris) }
func (ris reposByLastcommit) Less(i, j int) bool { return ris[i].lastcommit > ris[j].lastcommit }
func (ris reposByLastcommit) Swap(i, j int)      { ris[i], ris[j] = ris[j], ris[i] }

type reposByURL []repoInfo

func (ris reposByURL) Len() int           { return len(ris) }
func (ris reposByURL) Less(i, j int) bool { return ris[i].url < ris[j].url }
func (ris reposByURL) Swap(i, j int)      { ris[i], ris[j] = ris[j], ris[i] }

/*
	data loading (simple lines reader)
*/

func loadUrls() []string {
	return file2lines(urlFile)
}

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
	return !strings.Contains(l, " ") && len(l) != 0 && !strings.Contains(l, "#")
}
