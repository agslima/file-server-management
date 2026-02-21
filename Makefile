
# Makefile
# Generate and optionally publish a project directory tree.

TREE_OUT      := docs/tree.md
TREE_EXCLUDES := backend/vendor
TREE_FLAGS    := -I $(TREE_EXCLUDES)

.PHONY: tree tree-check tree-open tree-commit tree-push tree-publish

## Generate $(TREE_OUT) with the current project directory structure.
tree:
	@echo "Generating $(TREE_OUT)..."
	@tree $(TREE_FLAGS) > $(TREE_OUT)
	@echo "Wrote $(TREE_OUT)"

## Fail if $(TREE_OUT) is out-of-date relative to the current tree output.
tree-check:
	@echo "Checking whether $(TREE_OUT) is up-to-date..."
	@tmp=$$(mktemp); \
	tree $(TREE_FLAGS) > $$tmp; \
	if cmp -s $$tmp $(TREE_OUT); then \
		echo "$(TREE_OUT) is up-to-date."; \
		rm -f $$tmp; \
	else \
		echo "$(TREE_OUT) is out-of-date. Run: make tree"; \
		rm -f $$tmp; \
		exit 1; \
	fi

## Open $(TREE_OUT) in a pager.
tree-open:
	@less -R $(TREE_OUT)

## Commit $(TREE_OUT) (if it changed). Usage: make tree-commit MSG="Update tree"
tree-commit:
	@msg="$${MSG:-Update project tree}"; \
	git add $(TREE_OUT); \
	if git diff --cached --quiet; then \
		echo "No changes to commit for $(TREE_OUT)."; \
	else \
		git commit -m "$$msg"; \
	fi

## Push current branch to origin.
tree-push:
	@git push

## One-shot: regenerate tree, commit it, and push it.
## Usage: make tree-publish MSG="Update tree"
tree-publish: tree tree-commit tree-push

.PHONY: snyk-report
snyk-report:
	./hack/snyk-report.sh $(target_branch)

# Run linter on the code
.PHONY: lint
lint: test-tools-image
	$(call run-in-test-client,make lint-local)

# Run linter on the code (local version)
.PHONY: lint-local
lint-local:
	golangci-lint --version
	cd file-engine && golangci-lint run --fix --verbose

.PHONY: lint-ui
lint-ui: test-tools-image
	$(call run-in-test-client,make lint-ui-local)

.PHONY: lint-ui-local
lint-ui-local:
	cd ui && yarn lint
