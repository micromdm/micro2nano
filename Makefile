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

%-$(VERSION).zip: %.exe
	rm -f $@
	zip $@ $<

%-$(VERSION).zip: %
	rm -f $@
	zip $@ $<

clean:
	rm -f cmdapi-* llorne-*

release: \
	$(foreach bin,$(CMDAPI),$(subst .exe,,$(bin))-$(VERSION).zip) \
	$(foreach bin,$(LLORNE),$(subst .exe,,$(bin))-$(VERSION).zip)

test:
	go test -v -cover -race ./...

.PHONY: my $(CMDAPI) $(LLORNE) clean release test
