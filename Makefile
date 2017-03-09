run:
	time go run go/downloader/main.go


crawl:
	time go run go/crawler/main.go

stats:
	go run go/stats/main.go


all: run crawl stats;