#!/usr/bin/env bash
set -euo pipefail

# ─── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
die()   { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# ─── Config ───────────────────────────────────────────────────────────────────
REQUIRED_GO_MAJOR=1
REQUIRED_GO_MINOR=22
BINARY_NAME="carryless"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo "  Carryless — deployment script"
echo "  =============================="
echo ""

# ─── Detect OS ────────────────────────────────────────────────────────────────
OS="$(uname -s)"
case "$OS" in
  Linux*)   PLATFORM="linux" ;;
  Darwin*)  PLATFORM="macos" ;;
  *)        die "Unsupported platform: $OS. Only Linux and macOS are supported." ;;
esac
info "Platform: $PLATFORM"

# ─── Package manager helpers ──────────────────────────────────────────────────
install_apt() {
  info "Installing $* via apt..."
  sudo apt-get update -qq
  sudo apt-get install -y "$@"
}

install_brew() {
  info "Installing $* via brew..."
  brew install "$@"
}

install_pkg() {
  if [[ "$PLATFORM" == "linux" ]]; then
    if command -v apt-get &>/dev/null; then
      install_apt "$@"
    elif command -v dnf &>/dev/null; then
      info "Installing $* via dnf..."
      sudo dnf install -y "$@"
    elif command -v yum &>/dev/null; then
      info "Installing $* via yum..."
      sudo yum install -y "$@"
    elif command -v pacman &>/dev/null; then
      info "Installing $* via pacman..."
      sudo pacman -Sy --noconfirm "$@"
    else
      die "No supported package manager found (apt, dnf, yum, pacman). Please install $* manually."
    fi
  elif [[ "$PLATFORM" == "macos" ]]; then
    if ! command -v brew &>/dev/null; then
      die "Homebrew is not installed. Install it from https://brew.sh and re-run this script."
    fi
    install_brew "$@"
  fi
}

# ─── Check / install: gcc ─────────────────────────────────────────────────────
info "Checking for gcc..."
if ! command -v gcc &>/dev/null; then
  warn "gcc not found — installing..."
  if [[ "$PLATFORM" == "linux" ]]; then
    install_pkg gcc
  else
    # On macOS, gcc comes from Xcode command-line tools
    info "Installing Xcode command-line tools (provides gcc)..."
    xcode-select --install 2>/dev/null || true
    until xcode-select -p &>/dev/null; do sleep 5; done
  fi
fi
ok "gcc: $(gcc --version | head -1)"

# ─── Check / install: SQLite dev libraries ────────────────────────────────────
info "Checking for SQLite development libraries..."
SQLITE_OK=false

if [[ "$PLATFORM" == "linux" ]]; then
  if pkg-config --exists sqlite3 2>/dev/null; then
    SQLITE_OK=true
  elif [[ -f /usr/include/sqlite3.h ]]; then
    SQLITE_OK=true
  fi
  if ! $SQLITE_OK; then
    warn "SQLite dev headers not found — installing..."
    if command -v apt-get &>/dev/null; then
      install_pkg libsqlite3-dev
    elif command -v dnf &>/dev/null; then
      sudo dnf install -y sqlite-devel
    elif command -v yum &>/dev/null; then
      sudo yum install -y sqlite-devel
    elif command -v pacman &>/dev/null; then
      sudo pacman -Sy --noconfirm sqlite
    else
      die "Cannot install SQLite dev libraries automatically. Please install libsqlite3-dev (or equivalent) and re-run."
    fi
  fi
elif [[ "$PLATFORM" == "macos" ]]; then
  if brew list sqlite &>/dev/null 2>&1 || [[ -f /usr/include/sqlite3.h ]]; then
    SQLITE_OK=true
  fi
  if ! $SQLITE_OK; then
    warn "SQLite not found — installing via brew..."
    install_brew sqlite
  fi
fi
ok "SQLite dev libraries: present"

# ─── Check / install: Go ──────────────────────────────────────────────────────
info "Checking for Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+..."

go_ok() {
  if ! command -v go &>/dev/null; then return 1; fi
  local version
  version="$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)"
  local major minor
  major="$(echo "$version" | cut -d. -f1)"
  minor="$(echo "$version" | cut -d. -f2)"
  [[ "$major" -gt "$REQUIRED_GO_MAJOR" ]] || \
    { [[ "$major" -eq "$REQUIRED_GO_MAJOR" ]] && [[ "$minor" -ge "$REQUIRED_GO_MINOR" ]]; }
}

if ! go_ok; then
  warn "Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+ not found — installing..."
  GO_VERSION="1.22.5"

  if [[ "$PLATFORM" == "linux" ]]; then
    GOARCH_DL="amd64"
    [[ "$(uname -m)" == "aarch64" ]] && GOARCH_DL="arm64"
    GO_TARBALL="go${GO_VERSION}.linux-${GOARCH_DL}.tar.gz"
    GO_URL="https://go.dev/dl/${GO_TARBALL}"
    info "Downloading Go from ${GO_URL}..."
    curl -fsSL "$GO_URL" -o "/tmp/${GO_TARBALL}"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
    rm -f "/tmp/${GO_TARBALL}"
    export PATH="/usr/local/go/bin:$PATH"
    # Persist to profile
    PROFILE_LINE='export PATH="/usr/local/go/bin:$PATH"'
    for f in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
      [[ -f "$f" ]] && grep -qF "$PROFILE_LINE" "$f" || echo "$PROFILE_LINE" >> "$f"
    done
  elif [[ "$PLATFORM" == "macos" ]]; then
    install_brew go
    eval "$(brew shellenv)"
  fi
fi

if ! go_ok; then
  die "Go installation failed. Please install Go ${REQUIRED_GO_MAJOR}.${REQUIRED_GO_MINOR}+ manually from https://go.dev/dl/ and re-run."
fi
ok "Go: $(go version)"

# ─── Download Go module dependencies ─────────────────────────────────────────
info "Downloading Go module dependencies..."
cd "$SCRIPT_DIR"
go mod download
go mod verify
ok "Dependencies downloaded and verified."

# ─── Build ────────────────────────────────────────────────────────────────────
info "Building ${BINARY_NAME}..."
CGO_ENABLED=1 go build -ldflags='-w -s' -o "${BINARY_NAME}" .
ok "Build complete: ${SCRIPT_DIR}/${BINARY_NAME}"

# ─── Create .env file if missing ─────────────────────────────────────────────
ENV_FILE="${SCRIPT_DIR}/.env"
if [[ ! -f "$ENV_FILE" ]]; then
  info "Creating default .env file..."
  cat > "$ENV_FILE" <<'EOF'
# Carryless configuration
# Uncomment and set values as needed.

# PORT=8080
# DATABASE_PATH=carryless.db
# ENVIRONMENT=production   # or: development
# LOG_LEVEL=info

# Optional — Mailgun email notifications
# MAILGUN_DOMAIN=your-domain.com
# MAILGUN_API_KEY=your-api-key
EOF
  ok ".env file created at ${ENV_FILE}"
else
  info ".env already exists — skipping creation."
fi

# ─── Optional: systemd service (Linux only) ───────────────────────────────────
setup_systemd() {
  local svc="carryless"
  local unit_file="/etc/systemd/system/${svc}.service"

  if systemctl is-active --quiet "$svc" 2>/dev/null; then
    info "Stopping existing ${svc} service..."
    sudo systemctl stop "$svc"
  fi

  info "Writing systemd unit: ${unit_file}"
  sudo tee "$unit_file" > /dev/null <<EOF
[Unit]
Description=Carryless — backpacking gear planner
After=network.target

[Service]
Type=simple
User=${USER}
WorkingDirectory=${SCRIPT_DIR}
EnvironmentFile=-${ENV_FILE}
ExecStart=${SCRIPT_DIR}/${BINARY_NAME}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

  sudo systemctl daemon-reload
  sudo systemctl enable "$svc"
  sudo systemctl start "$svc"
  ok "Service started. Check status with: sudo systemctl status ${svc}"
  ok "View logs with: sudo journalctl -u ${svc} -f"
}

# ─── Run ──────────────────────────────────────────────────────────────────────
echo ""
echo "  Build successful!"
echo ""
echo "  To start the app, run:"
echo "    cd ${SCRIPT_DIR} && ./${BINARY_NAME}"
echo ""
echo "  Or with a custom port:"
echo "    PORT=3000 ./${BINARY_NAME}"
echo ""

if [[ "$PLATFORM" == "linux" ]] && command -v systemctl &>/dev/null; then
  read -r -p "  Set up a systemd service so Carryless starts on boot? [y/N] " response
  if [[ "${response,,}" == "y" ]]; then
    setup_systemd
    echo ""
    ok "Done! Carryless is running as a system service."
  else
    echo ""
    info "Skipped systemd setup. Start the app manually with: ./${BINARY_NAME}"
  fi
else
  info "Starting Carryless now..."
  echo ""
  # Load .env manually if it exists
  if [[ -f "$ENV_FILE" ]]; then
    set -o allexport
    # shellcheck disable=SC1090
    source <(grep -v '^#' "$ENV_FILE" | grep -v '^$')
    set +o allexport
  fi
  exec "${SCRIPT_DIR}/${BINARY_NAME}"
fi
