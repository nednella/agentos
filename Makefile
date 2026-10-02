VERSION ?= dev
LDFLAGS := -X github.com/nednella/agentos/internal/version.Version=$(VERSION)

build:
	go build -ldflags '$(LDFLAGS)' -o bin/agentos .

install:
	go install -ldflags '$(LDFLAGS)' .

test:
	go test ./...

fmt:
	gofmt -w .

desktop-frontend:
	cd desktop/frontend && (npm ci || npm install) && npm run build

desktop: desktop-frontend
	CGO_LDFLAGS='-framework UniformTypeIdentifiers' go build -tags desktop,production -ldflags '-w -s $(LDFLAGS)' -o bin/agentos-desktop ./desktop

desktop-run: desktop
	./bin/agentos-desktop

APP := bin/agentos.app

desktop-app: desktop
	rm -rf $(APP) bin/icon.iconset
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources bin/icon.iconset
	cp bin/agentos-desktop $(APP)/Contents/MacOS/agentos
	cp desktop/Info.plist $(APP)/Contents/Info.plist
	go run desktop/icon/gen.go bin/icon.png
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s bin/icon.png --out bin/icon.iconset/icon_$${s}x$${s}.png >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) bin/icon.png --out bin/icon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns bin/icon.iconset -o $(APP)/Contents/Resources/agentos.icns
	rm -rf bin/icon.iconset bin/icon.png
	codesign --force --sign - $(APP)

desktop-install: desktop-app
	@dest=/Applications; [ -w "$$dest" ] || { dest="$$HOME/Applications"; mkdir -p "$$dest"; }; \
	rm -rf "$$dest/agentos.app" && cp -R $(APP) "$$dest/" && echo "installed $$dest/agentos.app"

clean:
	rm -rf bin

.PHONY: build install test fmt desktop-frontend desktop desktop-run desktop-app desktop-install clean
