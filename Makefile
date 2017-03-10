run:
	time go run go/downloader/main.go


crawl:
	time go run go/crawler/main.go

stats:
	go run go/stats/main.go

sort:
	ruby go/sort.rb

all: sort run crawl stats;
