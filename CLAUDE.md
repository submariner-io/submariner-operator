# submariner-operator

Development guidelines for the submariner-operator repository.

## Commit Messages

@.agents/commit-templates.md

## Workflows

### Testing

#### Markdown

Run after editing any `.md` file, before committing:

```bash
make markdownlint
```

### CVE Fixes

@.agents/workflows/cve-fix.md

### Diagnosing and Fixing Konflux CI Failures

@.agents/workflows/konflux-ci-fix.md

### Adding Konflux Builds for the Operator on a New Branch

@.agents/workflows/konflux-branch-setup.md

### Adding Konflux Builds for the Bundle on a New Branch

@.agents/workflows/konflux-bundle-setup.md
