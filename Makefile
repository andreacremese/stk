VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build
build: ## Build the binary with version injected from git tag
	go build $(LDFLAGS) -o stk .

.PHONY: install
install: ## Install the binary with version injected from git tag
	go install $(LDFLAGS) .

# Install tools required by pre-commit hooks
.PHONY: install-hooks-tools
install-hooks-tools: ## Install pre-commit, semgrep, ggshield, and set up hooks if not present
	if ! command -v pre-commit >/dev/null 2>&1; then \
	  brew install pre-commit || echo "[WARNING] Could not install pre-commit with brew. Please install manually if needed."; \
	else \
	  echo "pre-commit already installed."; \
	fi
	if ! command -v semgrep >/dev/null 2>&1; then \
	  brew install semgrep || echo "[WARNING] Could not install semgrep with brew. Please install manually if needed."; \
	else \
	  echo "semgrep already installed."; \
	fi
	if ! command -v ggshield >/dev/null 2>&1; then \
	  brew install gitguardian/tap/ggshield || echo "[WARNING] Could not install ggshield with brew. Please install manually if needed."; \
	else \
	  echo "ggshield already installed."; \
	fi
	pre-commit install
	echo "pre-commit, semgrep, and ggshield installed (if needed). Pre-commit hooks installed."
