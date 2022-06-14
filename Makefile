VERSION = $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
OSARCH=$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)

CMDAPI=\
	cmdapi-darwin-amd64 \
	cmdapi-darwin-arm64 \
	cmdapi-linux-amd64

LLORNE=\
	llorne-darwin-amd64 \
	llorne-darwin-arm64 \
	llorne-linux-amd64

my: cmdapi-$(OSARCH) llorne-$(OSARCH)

$(CMDAPI): cmd/cmdapi
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

$(LLORNE): cmd/llorne
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

micro2nano-%-$(VERSION).zip: cmdapi-%.exe llorne-%.exe
	rm -rf $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	ln $^ $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

micro2nano-%-$(VERSION).zip: cmdapi-% llorne-%
	rm -rf $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	ln $^ $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

clean:
	rm -f cmdapi-* llorne-*

release: \
	micro2nano-darwin-amd64-$(VERSION).zip \
	micro2nano-darwin-arm64-$(VERSION).zip \
	micro2nano-linux-amd64-$(VERSION).zip

test:
	go test -v -cover -race ./...

.PHONY: my $(CMDAPI) $(LLORNE) clean release test
