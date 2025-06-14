BINARY = rune 
RUN_FILE = program.rn

.PHONY: all
all: build test

.PHONY: build
build:
	go build -o $(BINARY)

.PHONY: test
test: build
	python3 test.py $(filter)

.PHONY: run
run: build
	./rune run $(RUN_FILE)

.PHONY: clean
clean:
	rm -f $(BINARY)
