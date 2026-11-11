BINARY = rune
RUN_FILE = program.rn
FUZZTIME ?= 30s

.PHONY: all
all: build test

.PHONY: build
build:
	go build -o $(BINARY)

.PHONY: test
test:
	go test . ./pkg/... $(if $(filter),-run 'TestScripts/$(filter)')

.PHONY: update
update:
	go test . ./pkg/rune -update

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
	rm -f $(BINARY) cover.out

.PHONY: bench
bench:
	go test ./pkg/rune -run ^$$ -bench . -benchmem
