#### Setting up Operator Build in Konflux on New Branch

**Prerequisites:**

- Configuration added in [konflux-release-data](https://gitlab.cee.redhat.com/releng/konflux-release-data) repo (VPN required, see `tenants-config/cluster/kflux-prd-rh02/tenants/submariner-tenant/CLAUDE.md`)
- Existing Konflux-configured branch to copy files from (e.g., `release-0.21`)

**Placeholders:**

- `<target-branch>`: Your target branch (e.g., `release-0.22`)
- `<X-Y>`: Version with dashes (e.g., `0-22`)

**Important:** Operator uses gomod-only prefetch (no RPM dependencies), has CPE labels (ACM version mapping), and has bundle component in same repo (see bundle workflow). File change filters in Step 8 separate operator vs bundle builds.

##### 1. Checkout Bot's PR Branch

Bot creates PRs on branches named `konflux-submariner-operator-<X-Y>`.

```bash
git checkout konflux-submariner-operator-<X-Y>
```

##### 2. Configure YAMLlint to Ignore Generated Directories

Add `.tekton` to yamllint ignore list (idempotent, preserves existing rules).

```bash
TARGET_VERSION=$(echo "<target-branch>" | grep -oP '(?<=release-0\.)\d+$')
[ -z "$TARGET_VERSION" ] && { echo "ERROR: Invalid branch format"; exit 1; }
PREV_VERSION=$((TARGET_VERSION - 1))

grep -q "\.tekton" .yamllint.yml || sed -i '/^ignore: |$/a\  .tekton' .yamllint.yml
git add .yamllint.yml
git commit -s -m "Configure yamllint to ignore .tekton"
```

**Note:** Operator doesn't have `.rpm-lockfiles` (no RPM dependencies).

##### 3. Add Konflux Dockerfile and Configure Tekton to Use It

```bash
# Formula: Submariner 0.X → ACM 2.(X-7), so 0.22 → 2.15
TARGET_VERSION=$(echo "<target-branch>" | grep -oP '(?<=release-0\.)\d+$')
[ -z "$TARGET_VERSION" ] && { echo "ERROR: Invalid branch format. Expected release-0.XX"; exit 1; }
PREV_VERSION=$((TARGET_VERSION - 1))
ACM_VERSION=$((TARGET_VERSION - 7))

git checkout origin/release-0.${PREV_VERSION} -- package/Dockerfile.submariner-operator.konflux
sed -i "s/release-0.${PREV_VERSION}/<target-branch>/g" package/Dockerfile.submariner-operator.konflux
sed -i "s/cpe=\"cpe:\/a:redhat:acm:[0-9.]*::el9\"/cpe=\"cpe:\/a:redhat:acm:2.${ACM_VERSION}::el9\"/" package/Dockerfile.submariner-operator.konflux
sed -i '/^RUN go generate pkg\/embeddedyamls\/generate.go$/d' package/Dockerfile.submariner-operator.konflux

sed -i 's|package/Dockerfile.submariner-operator|package/Dockerfile.submariner-operator.konflux|g' .tekton/*.yaml
git add package/Dockerfile.submariner-operator.konflux .tekton/*.yaml
git commit -s -m "Add Konflux dockerfile for operator and configure tekton to use it"
```

**Note:** The `pkg/embeddedyamls` framework was removed upstream (commit 7b8ef030, July 2025). The sed command removes the obsolete `go generate` line that causes hermetic builds to fail.

##### 4. Add Build Args File

```bash
TARGET_VERSION=$(echo "<target-branch>" | grep -oP '(?<=release-0\.)\d+$')
[ -z "$TARGET_VERSION" ] && { echo "ERROR: Invalid branch format. Expected release-0.XX"; exit 1; }
PREV_VERSION=$((TARGET_VERSION - 1))

git checkout origin/release-0.${PREV_VERSION} -- .tekton/konflux.args
sed -i "s/release-0.${PREV_VERSION}/<target-branch>/g" .tekton/konflux.args
sed -i '/value: package\/Dockerfile.submariner-operator.konflux$/a\  - name: build-args-file\n    value: .tekton/konflux.args' .tekton/*.yaml
git add .tekton/konflux.args .tekton/*.yaml
git commit -s -m "Add build args file to tekton config"
```

##### 5. Enable Hermetic Builds

```bash
# Only add if not already present (idempotent)
if ! grep -q "^  - name: hermetic$" .tekton/*.yaml; then
  sed -i '/value: \.tekton\/konflux\.args$/a\  - name: prefetch-input\n    value: '\''[{"type": "gomod", "path": "."}, {"type": "gomod", "path": "tools"}]'\''\n  - name: hermetic\n    value: "true"' .tekton/*.yaml
fi
git add .tekton/*.yaml
git commit -s -m "Enable hermetic builds with gomod prefetching"
```

##### 6. Add Multi-Platform Support

```bash
if ! grep -q "linux/arm64" .tekton/*.yaml; then
  sed -i '/^    - linux\/x86_64$/a\    - linux/arm64\n    - linux/ppc64le\n    - linux/s390x' .tekton/*.yaml
  git add .tekton/*.yaml
  git commit -s -m "Add multi-platform build support"
fi
```

##### 7. Enable SBOM Generation

```bash
# Only add if not already present (idempotent)
if ! grep -q "^  - name: build-source-image$" .tekton/*.yaml; then
  sed -i '/  - name: hermetic$/,/    value: "true"$/{/    value: "true"$/a\  - name: build-source-image\n    value: "true"
}' .tekton/*.yaml
fi
git add .tekton/*.yaml
git commit -s -m "Enable SBOM generation"
```

##### 8. Add File Change Filters

Manually edit both `.tekton/*.yaml` files. Add path filters to the existing CEL expression.

**Note:** Examples below show `release-0.22` and `submariner-operator-0-22`. Substitute your actual target branch and tekton filenames.

Before:
```yaml
pipelinesascode.tekton.dev/on-cel-expression: event == "pull_request" && target_branch
  == "release-0.22"
```

After (substitute your branch and filenames):
```yaml
pipelinesascode.tekton.dev/on-cel-expression: event == "pull_request" && target_branch
  == "release-0.22" && (".tekton/submariner-operator-0-22-pull-request.yaml".pathChanged()
  || ".tekton/submariner-operator-0-22-push.yaml".pathChanged() || "go.mod".pathChanged()
  || "api/***".pathChanged() || "package/***".pathChanged() || "internal/***".pathChanged()
  || "pkg/***".pathChanged())
```

Do the same for the push file (change `event == "pull_request"` to `event == "push"`).

Verify:
```bash
yq eval '.' .tekton/*.yaml > /dev/null
```

Commit:
```bash
git add .tekton/*.yaml
git commit -s -m "Avoid building operator when updating bundle"
```

**Note:** Filters separate operator vs bundle builds in same repo.

##### 9. Update Task References

```bash
bash << 'EOF'
set -e

PATCHER_SHA="b001763bb1cd0286a894cfb570fe12dd7f4504bd"
EXPECTED_SHA256="080ad5d7cf7d0cee732a774b7e4dda0e2ccf26b58e08a8516a3b812bc73beb53"

SCRIPT=$(curl -sL "https://raw.githubusercontent.com/simonbaird/konflux-pipeline-patcher/${PATCHER_SHA}/pipeline-patcher")
ACTUAL_SHA256=$(echo "$SCRIPT" | sha256sum | cut -d' ' -f1)

if [[ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]]; then
  echo "ERROR: Script checksum mismatch!"
  exit 1
fi

echo "$SCRIPT" | bash -s bump-task-refs
EOF
git diff --quiet .tekton/*.yaml || { git add .tekton/*.yaml && git commit -s -m "Update Tekton task references to latest versions"; }
```

**Note:** Updates task references if outdated.

##### 10. Review and Push

```bash
git log origin/<target-branch>..HEAD
git status
git push origin konflux-submariner-operator-<X-Y> --force-with-lease
```

Expected: 8-9 commits (bot's initial + 7-8 from steps 2-9), clean working tree.

**Troubleshooting:**

- **Steps 5-7 skipped**: Bot may have already added hermetic, ARM64, or SBOM parameters. Expected.
- **Step 9 no commit**: Task references already up-to-date. Expected.
- **CPE version wrong**: Verify ACM version formula: 0.X → 2.(X-7). For example, 0.22 → 2.15.
