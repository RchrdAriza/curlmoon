#!/usr/bin/env sh
# curlmoon installer — checks prerequisites, detects platform, builds and installs.
#
# Usage:
#   ./install.sh              # build and install
#   PREFIX=/custom ./install.sh   # override install prefix
#   ./install.sh --uninstall  # remove the installed binary
set -eu

# --- constants -------------------------------------------------------------
BIN_NAME="curlmoon"
REPO_URL="${CURLMOON_REPO:-https://github.com/RchrdAriza/curlmoon.git}"
MIN_GO_MAJOR=1
MIN_GO_MINOR=24
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" 2>/dev/null && pwd || echo "$PWD")
CLONE_TMP=""
cleanup() { [ -n "$CLONE_TMP" ] && rm -rf "$CLONE_TMP"; return 0; }
trap cleanup EXIT INT TERM

# --- pretty output ---------------------------------------------------------
if [ -t 1 ]; then
	C_RESET=$(printf '\033[0m'); C_RED=$(printf '\033[31m')
	C_GREEN=$(printf '\033[32m'); C_YELLOW=$(printf '\033[33m')
	C_BLUE=$(printf '\033[34m'); C_BOLD=$(printf '\033[1m')
else
	C_RESET=; C_RED=; C_GREEN=; C_YELLOW=; C_BLUE=; C_BOLD=
fi
info()  { printf '%s==>%s %s\n' "$C_BLUE"  "$C_RESET" "$*"; }
ok()    { printf '%s ok %s %s\n' "$C_GREEN" "$C_RESET" "$*"; }
warn()  { printf '%swarn%s %s\n' "$C_YELLOW" "$C_RESET" "$*"; }
die()   { printf '%serr %s %s\n' "$C_RED"   "$C_RESET" "$*" >&2; exit 1; }

# --- platform detection ----------------------------------------------------
OS=$(uname -s 2>/dev/null || echo unknown)
ARCH=$(uname -m 2>/dev/null || echo unknown)
IS_TERMUX=0
[ -n "${PREFIX:-}" ] && [ -d "${PREFIX:-}/bin" ] && case "$PREFIX" in
	*com.termux*) IS_TERMUX=1 ;;
esac

# Map uname arch to Go's GOARCH, mostly for the user-facing summary.
case "$ARCH" in
	x86_64|amd64)          GOARCH_HINT=amd64 ;;
	aarch64|arm64)         GOARCH_HINT=arm64 ;;
	armv7l|armv6l|arm)     GOARCH_HINT=arm ;;
	i386|i686)             GOARCH_HINT=386 ;;
	*)                     GOARCH_HINT=$ARCH ;;
esac

# --- install destination ---------------------------------------------------
# Termux exports $PREFIX (e.g. /data/data/com.termux/files/usr). Elsewhere fall
# back to /usr/local/bin, or ~/.local/bin when we can't write there.
if [ "$IS_TERMUX" = 1 ]; then
	INSTALL_DIR="$PREFIX/bin"
elif [ -n "${PREFIX:-}" ]; then
	INSTALL_DIR="$PREFIX/bin"
elif [ -w /usr/local/bin ] 2>/dev/null; then
	INSTALL_DIR="/usr/local/bin"
elif [ "$(id -u 2>/dev/null || echo 1000)" = "0" ]; then
	INSTALL_DIR="/usr/local/bin"
else
	INSTALL_DIR="$HOME/.local/bin"
fi
DEST="$INSTALL_DIR/$BIN_NAME"

# --- uninstall -------------------------------------------------------------
if [ "${1:-}" = "--uninstall" ] || [ "${1:-}" = "-u" ]; then
	if [ -e "$DEST" ]; then
		rm -f "$DEST" && ok "removed $DEST"
	else
		warn "nothing to remove at $DEST"
	fi
	exit 0
fi

# --- checks ----------------------------------------------------------------
info "Platform: $C_BOLD$OS/$ARCH$C_RESET (GOARCH $GOARCH_HINT)$([ "$IS_TERMUX" = 1 ] && echo ' · Termux')"

command -v git >/dev/null 2>&1 && ok "git found" || warn "git not found (only needed to clone the repo, not to build)"

command -v go >/dev/null 2>&1 || die "Go toolchain not found. Install it: $([ "$IS_TERMUX" = 1 ] && echo 'pkg install golang' || echo 'https://go.dev/dl/')"

# Verify Go >= MIN_GO_MAJOR.MIN_GO_MINOR
GO_VER=$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')
GO_NUM=${GO_VER#go}
GO_MAJOR=$(echo "$GO_NUM" | cut -d. -f1)
GO_MINOR=$(echo "$GO_NUM" | cut -d. -f2)
[ -n "$GO_MAJOR" ] && [ -n "$GO_MINOR" ] || die "could not parse Go version from '$GO_VER'"
if [ "$GO_MAJOR" -lt "$MIN_GO_MAJOR" ] || { [ "$GO_MAJOR" -eq "$MIN_GO_MAJOR" ] && [ "$GO_MINOR" -lt "$MIN_GO_MINOR" ]; }; then
	die "Go $MIN_GO_MAJOR.$MIN_GO_MINOR+ required, found $GO_NUM"
fi
ok "Go $GO_NUM"

# Locate the source. When run from a checkout, build in place. When piped
# (curl | sh), there's no repo next to the script — clone it to a temp dir.
if [ -f "$SCRIPT_DIR/go.mod" ] && [ -d "$SCRIPT_DIR/cmd/$BIN_NAME" ]; then
	SRC_DIR="$SCRIPT_DIR"
else
	command -v git >/dev/null 2>&1 || die "git is required to fetch the source. Install it first."
	CLONE_TMP=$(mktemp -d 2>/dev/null || mktemp -d -t curlmoon)
	info "fetching source into $CLONE_TMP ..."
	git clone --depth 1 "$REPO_URL" "$CLONE_TMP/curlmoon" >/dev/null 2>&1 \
		|| die "git clone failed ($REPO_URL)"
	SRC_DIR="$CLONE_TMP/curlmoon"
fi

# --- install dir ready ------------------------------------------------------
if [ ! -d "$INSTALL_DIR" ]; then
	info "creating $INSTALL_DIR"
	mkdir -p "$INSTALL_DIR" || die "could not create $INSTALL_DIR"
fi
[ -w "$INSTALL_DIR" ] || die "no write permission for $INSTALL_DIR (try sudo, or set PREFIX=\$HOME/.local)"

# --- build & install --------------------------------------------------------
info "building $BIN_NAME ..."
( cd "$SRC_DIR" && go build -o "$DEST" "./cmd/$BIN_NAME" ) || die "build failed"
ok "installed to $C_BOLD$DEST$C_RESET"

# --- PATH hint --------------------------------------------------------------
case ":$PATH:" in
	*":$INSTALL_DIR:"*) : ;;
	*) warn "$INSTALL_DIR is not on your PATH — add it, e.g.:"
	   printf '      export PATH="%s:$PATH"\n' "$INSTALL_DIR" ;;
esac

printf '\n%sDone.%s Run %s%s%s to start.\n' "$C_GREEN" "$C_RESET" "$C_BOLD" "$BIN_NAME" "$C_RESET"
