BINARY := bin/agent
CMD := ./cmd/agent

BINARY_KUBENPUCTL := bin/kubenpuctl
CMD_KUBENPUCTL := ./cmd/kubenpuctl


GOOS := linux
GOARCH := amd64

.PHONY: build build-agent build-kubenpuctl generate test clean vet verify

build: build-agent build-kubenpuctl

build-agent:
	mkdir -p bin
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY) $(CMD)

build-kubenpuctl:
	mkdir -p bin
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_KUBENPUCTL) $(CMD_KUBENPUCTL)

generate:
	go generate ./...

test:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go test ./... -v

vet:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go vet ./...

clean:
	rm -rf bin

verify:
	bpftool prog load pkg/loader/bpf_x86_bpfel.o /sys/fs/bpf/agent_test
	rm /sys/fs/bpf/agent_test

