#!/usr/bin/env bash
#
# Setup script for git hooks and development tools
# Usage: ./scripts/setup-hooks.sh
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

echo ""
echo "=========================================="
echo "  pgn-extract-go Development Setup"
echo "=========================================="
echo ""

# =============================================================================
# Option selection
# =============================================================================
echo "Choose hook installation method:"
echo ""
echo "  1) pre-commit framework (recommended)"
echo "     - Requires Python and pip"
echo "     - More features, better maintained"
echo "     - Automatic updates"
echo ""
echo "  2) Native git hooks"
echo "     - No external dependencies"
echo "     - Simpler setup"
echo "     - Manual updates"
echo ""
read -rp "Enter choice [1/2]: " CHOICE

case $CHOICE in
    1)
        info "Setting up pre-commit framework..."

        # Check if pre-commit is installed
        if ! command -v pre-commit &> /dev/null; then
            info "Installing pre-commit..."
            if command -v pip3 &> /dev/null; then
                pip3 install pre-commit
            elif command -v pip &> /dev/null; then
                pip install pre-commit
            elif command -v brew &> /dev/null; then
                brew install pre-commit
            else
                error "Cannot install pre-commit. Please install pip or use option 2."
                exit 1
            fi
        fi

        # Install pre-commit hooks
        pre-commit install
        pre-commit install --hook-type pre-push

        success "pre-commit hooks installed"

        # Run pre-commit on all files to verify
        info "Running pre-commit on all files (first run may take a while)..."
        pre-commit run --all-files || true
        ;;

    2)
        info "Setting up native git hooks..."

        # Create symlinks for hooks
        HOOKS_DIR="$(git rev-parse --git-dir)/hooks"

        # Pre-commit hook
        if [ -f "$HOOKS_DIR/pre-commit" ]; then
            warn "Existing pre-commit hook found, backing up..."
            mv "$HOOKS_DIR/pre-commit" "$HOOKS_DIR/pre-commit.bak"
        fi
        ln -sf "../../scripts/pre-commit" "$HOOKS_DIR/pre-commit"
        chmod +x "$HOOKS_DIR/pre-commit"
        success "pre-commit hook installed"

        # Pre-push hook
        if [ -f "$HOOKS_DIR/pre-push" ]; then
            warn "Existing pre-push hook found, backing up..."
            mv "$HOOKS_DIR/pre-push" "$HOOKS_DIR/pre-push.bak"
        fi
        ln -sf "../../scripts/pre-push" "$HOOKS_DIR/pre-push"
        chmod +x "$HOOKS_DIR/pre-push"
        success "pre-push hook installed"
        ;;

    *)
        error "Invalid choice"
        exit 1
        ;;
esac

echo ""

# =============================================================================
# Install Go tools
# =============================================================================
info "Checking Go development tools..."

TOOLS_TO_INSTALL=()

if ! command -v golangci-lint &> /dev/null; then
    TOOLS_TO_INSTALL+=("golangci-lint")
fi

if ! command -v staticcheck &> /dev/null; then
    TOOLS_TO_INSTALL+=("staticcheck")
fi

if ! command -v goimports &> /dev/null; then
    TOOLS_TO_INSTALL+=("goimports")
fi

if [ ${#TOOLS_TO_INSTALL[@]} -gt 0 ]; then
    echo ""
    echo "The following Go tools are recommended but not installed:"
    for tool in "${TOOLS_TO_INSTALL[@]}"; do
        echo "  - $tool"
    done
    echo ""
    read -rp "Install missing tools? [y/N]: " INSTALL_TOOLS

    if [[ "$INSTALL_TOOLS" =~ ^[Yy]$ ]]; then
        for tool in "${TOOLS_TO_INSTALL[@]}"; do
            case $tool in
                "golangci-lint")
                    info "Installing golangci-lint..."
                    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v1.62.2
                    success "golangci-lint installed"
                    ;;
                "staticcheck")
                    info "Installing staticcheck..."
                    go install honnef.co/go/tools/cmd/staticcheck@latest
                    success "staticcheck installed"
                    ;;
                "goimports")
                    info "Installing goimports..."
                    go install golang.org/x/tools/cmd/goimports@latest
                    success "goimports installed"
                    ;;
            esac
        done
    fi
else
    success "All Go tools are already installed"
fi

echo ""
echo "=========================================="
success "Setup complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "  1. Make changes to the code"
echo "  2. Stage your changes: git add ."
echo "  3. Commit: git commit -m 'Your message'"
echo "     (pre-commit hooks will run automatically)"
echo ""
echo "To run checks manually:"
echo "  pre-commit run -a"
echo ""
echo "To update hooks:"
echo "  pre-commit autoupdate"
echo ""
