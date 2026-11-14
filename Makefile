VERSION ?= dev
LDFLAGS := -X github.com/nednella/agentos/internal/version.Version=$(VERSION:v%=%)

test:
	go test ./...

fmt:
	gofmt -w .

desktop-frontend:
	cd desktop/frontend && (npm ci || npm install) && npm run build

desktop: desktop-frontend
	CGO_LDFLAGS='-framework UniformTypeIdentifiers' go build -tags desktop,production -ldflags '-w -s $(LDFLAGS)' -o bin/agentos-desktop ./desktop

APP := bin/agentos.app

# Sessions drive their browser tab with agent-browser. The version is pinned: agentos binds each session to its tab
# through a file of this version's daemon (desktop/browser/agentbrowser.go), so check that binding before a bump.
AGENT_BROWSER_VERSION := 0.43.0
AGENT_BROWSER_SHA256 := 05338a8eeddf0b3f9ebf0c87a6c21aedc10685608c68618e1a8483c43eae9c3f
AGENT_BROWSER := bin/agent-browser-$(AGENT_BROWSER_VERSION)

$(AGENT_BROWSER):
	mkdir -p bin
	curl -fsSL https://github.com/vercel-labs/agent-browser/releases/download/v$(AGENT_BROWSER_VERSION)/agent-browser-darwin-arm64 -o $@.part
	echo "$(AGENT_BROWSER_SHA256)  $@.part" | shasum -a 256 -c - >/dev/null
	chmod +x $@.part
	mv $@.part $@

# bin/agent-browser is the pinned binary for tests: AGENTOS_AGENT_BROWSER=bin/agent-browser.
agent-browser: $(AGENT_BROWSER)
	cp $(AGENT_BROWSER) bin/agent-browser

# A release signs with a stable certificate so macOS keeps its permission grants across updates; "-" signs ad-hoc.
SIGN_IDENTITY ?= -

# A local build gets its own name so it never passes for the installed agentos.app.
desktop-app: bin/agentos-dev.app

bin/%.app: desktop $(AGENT_BROWSER)
	rm -rf $@ bin/icon.iconset
	mkdir -p $@/Contents/MacOS $@/Contents/Helpers $@/Contents/Resources bin/icon.iconset
	cp bin/agentos-desktop $@/Contents/MacOS/agentos
	cp $(AGENT_BROWSER) $@/Contents/Helpers/agent-browser
	cp desktop/build/Info.plist $@/Contents/Info.plist
	plutil -replace CFBundleShortVersionString -string '$(VERSION:v%=%)' $@/Contents/Info.plist
	go run desktop/build/icon.go bin/icon.png
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s bin/icon.png --out bin/icon.iconset/icon_$${s}x$${s}.png >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) bin/icon.png --out bin/icon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns bin/icon.iconset -o $@/Contents/Resources/agentos.icns
	rm -rf bin/icon.iconset bin/icon.png
	codesign --force --sign "$(SIGN_IDENTITY)" $@/Contents/Helpers/agent-browser
	codesign --force --sign "$(SIGN_IDENTITY)" $@

# The app and the command are one binary: the command is a link into the bundle.
desktop-install: $(APP)
	mkdir -p "$(HOME)/Applications" "$(HOME)/.local/bin"
	rm -rf "$(HOME)/Applications/agentos.app" && cp -R $(APP) "$(HOME)/Applications/"
	ln -sfn "$(HOME)/Applications/agentos.app/Contents/MacOS/agentos" "$(HOME)/.local/bin/agentos"
	@echo "installed $(HOME)/Applications/agentos.app; agentos is $(HOME)/.local/bin/agentos"

RELEASE := bin/agentos-darwin-arm64.zip

release: $(APP)
	rm -f $(RELEASE) $(RELEASE).sha256
	ditto -c -k --keepParent $(APP) $(RELEASE)
	cd bin && shasum -a 256 agentos-darwin-arm64.zip > agentos-darwin-arm64.zip.sha256

clean:
	rm -rf bin

.PHONY: test fmt desktop-frontend desktop desktop-app desktop-install release clean agent-browser
