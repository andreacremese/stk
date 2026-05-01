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
install-hooks-tools: ## Install pre-commit (via uv), semgrep, ggshield, and set up hooks if not present
	if ! command -v uv >/dev/null 2>&1; then \
	  echo "[ERROR] uv is required to install pre-commit. Install it from https://docs.astral.sh/uv/"; exit 1; \
	fi
	uv tool install pre-commit --python python3.13
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
	echo "pre-commit (via uv/python3.13), semgrep, and ggshield installed. Pre-commit hooks installed."
