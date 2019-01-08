cmd := $(shell ls cmd/)

all: $(cmd)

GitHash := github.com/dearcode/doodle/util.GitHash
GitTime := github.com/dearcode/doodle/util.GitTime
GitMessage := github.com/dearcode/doodle/util.GitMessage


LDFLAGS += -X "$(GitHash)=$(shell git log --pretty=format:'%H' -1)"
LDFLAGS += -X "$(GitTime)=$(shell git log --pretty=format:'%cd' -1)"
LDFLAGS += -X "$(GitMessage)=$(shell git log --pretty=format:'%cn %s %b' -1)"

source := $(shell ls -ld */|awk '$$NF !~ /bin\/|logs\/|config\/|_vendor\/|vendor\/|web\/|Godeps\/|docs\// {printf $$NF" "}')

golint:
	go get github.com/golang/lint/golint

megacheck:
	go get honnef.co/go/tools/cmd/megacheck

lint: golint megacheck
	for path in $(source); do golint "$$path..."; done;
	for path in $(source); do gofmt -s -l -w $$path;  done;
	go tool vet $(source) 2>&1
	megacheck ./...



clean:
	@rm -rf bin

.PHONY: $(cmd)

$(cmd):
	go build -o bin/$@ -ldflags '$(LDFLAGS)' cmd/$@/main.go 


test:
	@for path in $(source); do echo "go test ./$$path"; go test "./"$$path; done;

