BINARY := bin/agent
CMD := ./cmd/agent

GOOS := linux
GOARCH := amd64

.PHONY: build generate test clean vet verify

build:
	mkdir -p bin
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY) $(CMD)

generate:
	go generate ./...

test:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go test ./...

vet:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go vet ./...

clean:
	rm -rf bin

verify:
	bpftool prog load pkg/loader/bpf_x86_bpfel.o /sys/fs/bpf/agent_test
	rm /sys/fs/bpf/agent_test
