#!/usr/bin/env bash
# Installs the latest agentos release: curl -fsSL https://raw.githubusercontent.com/nednella/agentos/main/install.sh | bash
set -euo pipefail

REPO="nednella/agentos"
ASSET="agentos-darwin-arm64.zip"
DOWNLOAD="https://github.com/$REPO/releases/latest/download/$ASSET"
APP_DIR="$HOME/Applications"
APP="$APP_DIR/agentos.app"
BIN_DIR="$HOME/.local/bin"
LINK="$BIN_DIR/agentos"
CHROME_APP="$APP_DIR/Google Chrome for Testing.app"
CHROME_LATEST="https://googlechromelabs.github.io/chrome-for-testing/LATEST_RELEASE_STABLE"
CHROME_STORE="https://storage.googleapis.com/chrome-for-testing-public"

if [ -t 1 ]; then
    CYAN=$'\033[96m' GREEN=$'\033[92m' YELLOW=$'\033[93m' RED=$'\033[91m' DIM=$'\033[2m' BOLD=$'\033[1m' RESET=$'\033[0m'
else
    CYAN="" GREEN="" YELLOW="" RED="" DIM="" BOLD="" RESET=""
fi

ok()   { echo "  ${GREEN}ok${RESET}    $1"; }
warn() { echo "  ${YELLOW}warn${RESET}  $1"; }
die()  { echo "  ${RED}error${RESET} $1" >&2; echo; exit 1; }

# The agent browser is Chrome for Testing, not the browser you use: macOS sends a clicked link to any running copy of
# the default browser's app, and a copy agentos runs has no window to show it in.
install_chrome() {
    local version have
    version=$(curl -fsSL "$CHROME_LATEST") || return 1
    have=$(plutil -extract CFBundleShortVersionString raw -o - "$CHROME_APP/Contents/Info.plist" 2>/dev/null || true)
    if [ "$have" != "$version" ]; then
        curl -fsSL "$CHROME_STORE/$version/mac-arm64/chrome-mac-arm64.zip" -o "$TMP/chrome.zip" || return 1
        ditto -x -k "$TMP/chrome.zip" "$TMP" || return 1
        rm -rf "$CHROME_APP"
        mv "$TMP/chrome-mac-arm64/Google Chrome for Testing.app" "$CHROME_APP" || return 1
    fi
    ok "Chrome for Testing $version, for the agent browser"
}

echo
echo "${CYAN}${BOLD}agentos${RESET} ${DIM}· one screen for all your coding agents${RESET}"
echo

[ "$(uname -s)" = "Darwin" ] || die "agentos runs on macOS only"
[ "$(uname -m)" = "arm64" ] || die "agentos needs a Mac with Apple silicon"
ok "macOS on Apple silicon"

TMP=$(mktemp -d "${TMPDIR:-/tmp}/agentos-install.XXXXXX")
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$DOWNLOAD" -o "$TMP/$ASSET" || die "could not download $DOWNLOAD"
curl -fsSL "$DOWNLOAD.sha256" -o "$TMP/$ASSET.sha256" || die "could not download $DOWNLOAD.sha256"
(cd "$TMP" && shasum -a 256 -c "$ASSET.sha256" >/dev/null 2>&1) || die "the download does not match its sha256"
ok "downloaded the latest release"

ditto -x -k "$TMP/$ASSET" "$TMP" || die "could not unpack $ASSET"
[ -x "$TMP/agentos.app/Contents/MacOS/agentos" ] || die "$ASSET holds no agentos.app"
mkdir -p "$APP_DIR"
rm -rf "$APP"
mv "$TMP/agentos.app" "$APP"
VERSION=$("$APP/Contents/MacOS/agentos" version | awk '{print $NF}')
ok "installed agentos v$VERSION to $APP"

mkdir -p "$BIN_DIR"
ln -sfn "$APP/Contents/MacOS/agentos" "$LINK"
ok "linked the agentos command to $LINK"

case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *) warn "$BIN_DIR is not on your PATH; add this to your shell profile: ${CYAN}export PATH=\"\$HOME/.local/bin:\$PATH\"${RESET}" ;;
esac

install_chrome || warn "could not install Chrome for Testing to $APP_DIR; agents get no browser until you run this script again"

if command -v tmux >/dev/null; then ok "tmux $(tmux -V | awk '{print $2}')"; else warn "tmux is missing; agents run inside it: ${CYAN}brew install tmux${RESET}"; fi
if command -v gh >/dev/null; then
    if gh auth status >/dev/null 2>&1; then ok "gh is logged in"; else warn "gh is not logged in; the queue reads your issues with it: ${CYAN}gh auth login${RESET}"; fi
else
    warn "gh is missing; the queue reads your issues with it: ${CYAN}brew install gh && gh auth login${RESET}"
fi
if command -v claude >/dev/null; then ok "claude $(claude --version 2>/dev/null | awk '{print $1}')"; else warn "Claude Code is missing; agentos starts sessions with it: https://docs.claude.com/en/docs/claude-code"; fi

if pgrep -qf "$APP/Contents/MacOS/agentos"; then
    warn "agentos is running the old version; quit it and open it again"
fi

echo
echo "  Run ${CYAN}agentos${RESET} to open it."
echo
