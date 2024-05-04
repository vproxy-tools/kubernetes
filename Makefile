.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "Usage: \
	\n    make clean \
	\n    make apiserver \
	"

.PHONY: clean
clean:
	rm -rf _output

_output:
	mkdir -p _output

.PHONY: apiserver
apiserver: _output
	go build -o _output/apiserver cmd/apiserver/apiserver.go
