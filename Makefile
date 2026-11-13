BINARY = rune
RUN_FILE = program.rn
FUZZTIME ?= 30s
VERSION ?= dev
LDFLAGS = -s -w -X main.version=$(VERSION)
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: all
all: build test

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY)

.PHONY: dist
dist:
	rm -rf dist
	for p in $(PLATFORMS); do \
		export GOOS=$${p%/*} GOARCH=$${p#*/} CGO_ENABLED=0; \
		name=rune-$(VERSION)-$$GOOS-$$GOARCH; \
		go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$$name/rune$$(go env GOEXE) . || exit 1; \
		cp LICENSE README.md dist/$$name; \
		tar -czf dist/$$name.tar.gz -C dist $$name; \
		rm -r dist/$$name; \
	done
	cd dist && shasum -a 256 *.tar.gz > checksums.txt

.PHONY: test
test:
	go test . ./pkg/... $(if $(filter),-run 'TestScripts/$(filter)')

.PHONY: update
update:
	go test ./... -update

.PHONY: cover
cover:
	go test . ./pkg/... -coverpkg ./... -coverprofile cover.out
	go tool cover -func cover.out | tail -1

.PHONY: fuzz
fuzz:
	go test ./pkg/rune -run '^$$' -fuzz FuzzRun -fuzztime $(FUZZTIME)

.PHONY: run
run: build
	./rune run $(RUN_FILE)

.PHONY: clean
clean:
	rm -rf $(BINARY) cover.out dist

.PHONY: bench
bench:
	go test ./pkg/rune -run ^$$ -bench . -benchmem
