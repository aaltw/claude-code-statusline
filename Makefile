BINARY := statusline-bin
INSTALL_DIR := $(HOME)/.claude

.PHONY: build install clean

build:
	cd go && go build -ldflags "-s -w" -o ../$(BINARY) .

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	chmod +x $(INSTALL_DIR)/$(BINARY)

clean:
	rm -f $(BINARY)
