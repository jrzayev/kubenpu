BINARY := bin/agent
CMD := ./cmd/agent

BINARY_KUBENPUCTL := bin/kubenpuctl
CMD_KUBENPUCTL := ./cmd/kubenpuctl


VERSION := $(shell tr -d '[:space:]' < VERSION)

GOOS := linux
GOARCH := amd64

.PHONY: build build-agent build-kubenpuctl generate test clean vet verify set-version check-version

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

fmt:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go fmt ./...

clean:
	rm -rf bin

verify:
	bpftool prog load pkg/loader/bpf_x86_bpfel.o /sys/fs/bpf/agent_test
	rm /sys/fs/bpf/agent_test


CHART := deploy/helm/Chart.yaml
KUSTOMIZATION := deploy/kustomize/base/kustomization.yaml

set-version:
	V=$(VERSION) yq -i '.version = strenv(V) | .appVersion = strenv(V) | .appVersion style="double"' $(CHART)
	V=$(VERSION) yq -i '.labels[0].pairs["app.kubernetes.io/version"] = strenv(V) | .labels[0].pairs["app.kubernetes.io/version"] style="double" | .images[0].newTag = strenv(V) | .images[0].newTag style="double"' $(KUSTOMIZATION)

check-version:
	@test "$$(yq '.version' $(CHART))" = "$(VERSION)" || { echo "Chart.yaml version != $(VERSION)"; exit 1; }
	@test "$$(yq '.appVersion' $(CHART))" = "$(VERSION)" || { echo "Chart.yaml appVersion != $(VERSION)"; exit 1; }
	@test "$$(yq '.labels[0].pairs["app.kubernetes.io/version"]' $(KUSTOMIZATION))" = "$(VERSION)" || { echo "kustomization.yaml label != $(VERSION)"; exit 1; }
	@test "$$(yq '.images[0].newTag' $(KUSTOMIZATION))" = "$(VERSION)" || { echo "kustomization.yaml newTag != $(VERSION)"; exit 1; }
	@echo "version $(VERSION) ok"
