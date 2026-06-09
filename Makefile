PREFIX ?= $(HOME)/bin
VERSION ?= 0.1.0
BINARY := wrapux
FISH_FUNCTIONS ?= $(HOME)/.config/fish/functions
FISH_FUNCTION := rmx.fish

.PHONY: build install uninstall fish test clean

build:
	go build -ldflags "-X wrapux/cmd.Version=$(VERSION)" -o $(BINARY) .

install: build fish
	mkdir -p $(PREFIX)
	cp $(BINARY) $(PREFIX)/$(BINARY)
	codesign --force --sign - $(PREFIX)/$(BINARY)

fish:
	mkdir -p $(FISH_FUNCTIONS)
	cp $(FISH_FUNCTION) $(FISH_FUNCTIONS)/$(FISH_FUNCTION)

uninstall:
	rm -f $(PREFIX)/$(BINARY)
	rm -f $(FISH_FUNCTIONS)/$(FISH_FUNCTION)

test:
	go test ./...

clean:
	rm -f $(BINARY)
