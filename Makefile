.PHONY: all clean

KITE_BIN     := go-kite
APP_NAME     := Kite
BUILDAPP_DIR := ../go-gui/cmd/buildapp
BUILDAPP_BIN := $(BUILDAPP_DIR)/buildapp

all: $(APP_NAME).app

$(KITE_BIN): *.go go.mod go.sum
	go build -o $@ .

$(BUILDAPP_BIN):
	cd $(BUILDAPP_DIR) && go build -o buildapp .

$(APP_NAME).app: $(KITE_BIN) $(BUILDAPP_BIN)
	$(BUILDAPP_BIN) -bundle-deps -o . -name $(APP_NAME) \
		-id github.com.go-gui-org.go-kite $(KITE_BIN)

clean:
	rm -f $(KITE_BIN)
	rm -rf $(APP_NAME).app
	cd $(BUILDAPP_DIR) && rm -f buildapp
