PREFIX ?= $(HOME)/bin
VERSION ?= 0.1.0
BINARY := wrapux

.PHONY: build install uninstall test clean

build:
	go build -ldflags "-X wrapux/cmd.Version=$(VERSION)" -o $(BINARY) .

install: build
	mkdir -p $(PREFIX)
	cp $(BINARY) $(PREFIX)/$(BINARY)
	codesign --force --sign - $(PREFIX)/$(BINARY)

uninstall:
	rm -f $(PREFIX)/$(BINARY)

test:
	go test ./...

clean:
	rm -f $(BINARY)
