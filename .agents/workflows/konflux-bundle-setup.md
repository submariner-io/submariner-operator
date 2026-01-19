#### Setting up Bundle Build in Konflux on New Branch

**Prerequisites:**

- Configuration added in konflux-ci/build-definitions repo
- Existing Konflux-configured bundle branch (to copy files from)

**Note:** `release-0.20` is the example source branch (any Konflux-enabled bundle branch works). For your target branch (e.g., `release-0.22`), replace `<target-branch>` in commands below.

##### 1. Checkout Bot's PR Branch

Bot creates PR with `.tekton/*.yaml` on branch named like `konflux-submariner-bundle-0-22`.

```bash
git checkout <bot-branch-name>  # e.g., konflux-submariner-bundle-0-22
```

Then run steps 2-11 below.

**Reference:**

- `5f1a8be4` - Red Hat Konflux kflux-prd-rh02 update submariner-bundle-0-22

##### 2. Add YAMLlint Ignore

**Note:** Usually already present from bot. Verify `.yamllint.yml` includes `.tekton` in ignore list.

```bash
# Verify it exists
grep -q "\.tekton" .yamllint.yml && echo "Already present" || {
  git checkout release-0.20 -- .yamllint.yml
  git add .yamllint.yml
  git commit -s -m "Ignore .tekton in YAMLlint"
}
```

##### 3. Add Konflux Bundle Infrastructure

Bundles need specific infrastructure (unlike operators which need Dockerfiles).

```bash
# Copy bundle infrastructure from existing branch
git checkout release-0.20 -- bundle.Dockerfile.konflux config/bundle/

# Update version references (0.20 → target version, 2.13 → target ACM version)
# For 0.22: 2.15 (0.X → 2.(X-7))
sed -i 's/0\.20/<target-version>/g; s/0-20/<target-version-dashed>/g; s/2\.13/<acm-version>/g' \
  bundle.Dockerfile.konflux config/bundle/kustomization.yaml config/bundle/patches/submariner.csv.config.yaml

# Update tekton to use Konflux bundle Dockerfile
sed -i 's|value: bundle.Dockerfile$|value: bundle.Dockerfile.konflux|' .tekton/submariner-bundle-*.yaml

# Update konflux.args
sed -i 's/BASE_BRANCH=.*/BASE_BRANCH=<target-branch>/' .tekton/konflux.args

git add -f bundle.Dockerfile.konflux config/bundle/ .tekton/
git commit -s -m "Add Konflux bundle infrastructure for <target-branch>"
```

**Reference:**

- `054ea753` - Add Konflux bundle infrastructure for release-0.22

##### 4. Verify Images Are Pinned

EC validation requires `registry.redhat.io/rhacm2/...@sha256:...` format:

```bash
grep -q "registry.redhat.io.*@sha256:" config/manager/patches/related-images.deployment.config.yaml && \
  echo "Images pinned - continue to step 5" || \
  echo "WARNING: Images need transformation"
```

If the check fails, the file needs transformation. It may have different variable names, `quay.io/...` URLs, and different components.

See @.agents/workflows/bundle-sha-update.md:
- Step 1: Correct variable names (component mapping) and Konflux snapshot SHAs
- Step 2: Target format (`registry.redhat.io/rhacm2/...@sha256:...`)

##### 5. Add OLM Feature Annotations

Add required OLM feature annotations to the CSV base template. These annotations are validated by Enterprise Contract.

```bash
# Check if annotations already exist
if ! grep -q "features.operators.openshift.io/disconnected" config/manifests/bases/submariner.clusterserviceversion.yaml; then
  # Add feature annotations after description line
  sed -i '/description: Creates and manages Submariner deployments./a\    features.operators.openshift.io/disconnected: "true"\n    features.operators.openshift.io/fips-compliant: "true"\n    features.operators.openshift.io/proxy-aware: "false"\n    features.operators.openshift.io/tls-profiles: "false"\n    features.operators.openshift.io/token-auth-aws: "false"\n    features.operators.openshift.io/token-auth-azure: "false"\n    features.operators.openshift.io/token-auth-gcp: "false"' config/manifests/bases/submariner.clusterserviceversion.yaml

  git add config/manifests/bases/submariner.clusterserviceversion.yaml
  git commit -s -m "Add required OLM feature annotations to CSV base"
fi

# Add subscription annotation
if ! grep -q "valid-subscription" config/manifests/bases/submariner.clusterserviceversion.yaml; then
  # Add after suggested-namespace line
  sed -i '/operatorframework.io\/suggested-namespace: submariner-operator/a\    operators.openshift.io/valid-subscription: '\''["OpenShift Platform Plus", "Red Hat\n      Advanced Cluster Management for Kubernetes"]'\''' config/manifests/bases/submariner.clusterserviceversion.yaml

  git add config/manifests/bases/submariner.clusterserviceversion.yaml
  git commit -s -m "Add required subscription annotation to CSV base"
fi
```

**Note:** These annotations indicate operator capabilities and requirements:
- `disconnected`: Works in disconnected/air-gapped environments
- `fips-compliant`: FIPS 140-2 compliant
- `proxy-aware`: Supports HTTP/HTTPS proxy configuration
- `tls-profiles`: Supports OpenShift TLS security profiles
- `token-auth-*`: Supports cloud provider token authentication
- `valid-subscription`: Required Red Hat subscriptions for this operator

**Reference:**

- `288af498` - Add required OLM feature annotations to bundle
- `661673d5` - Add required subscription annotation to bundle

##### 6. Add Build Args File

```bash
# Verify build-args-file parameter is set in spec.params
if ! awk '/^spec:/,/^  pipelineSpec:/' .tekton/submariner-bundle-*-pull-request.yaml | grep -q "name: build-args-file"; then
  sed -i '/value: bundle.Dockerfile.konflux$/a\  - name: build-args-file\n    value: .tekton/konflux.args' .tekton/submariner-bundle-*.yaml
  git add .tekton/submariner-bundle-*.yaml
  git commit -s -m "Add build args file to bundle tekton config"
fi
```

##### 7. Enable Hermetic Builds and SBOM

```bash
# Add hermetic and build-source-image parameters
sed -i '/value: \.tekton\/konflux\.args$/a\  - name: hermetic\n    value: "true"\n  - name: build-source-image\n    value: "true"' .tekton/submariner-bundle-*.yaml
git add .tekton/submariner-bundle-*.yaml
git commit -s -m "Enable hermetic builds and SBOM for bundle"
```

**Note:** Bundles don't use prefetch-input (no go modules to prefetch).

##### 8. Add Multi-Platform Support

```bash
sed -i '/value: bundle.Dockerfile.konflux$/a\  - name: build-platforms\n    value:\n    - linux/x86_64\n    - linux/ppc64le\n    - linux/s390x\n    - linux/arm64' .tekton/submariner-bundle-*.yaml
git add .tekton/submariner-bundle-*.yaml
git commit -s -m "Add multi-platform build support to bundle"
```

##### 9. Add File Change Filters

Manually edit both `.tekton/submariner-bundle-*.yaml` files. Add path filters to the existing CEL expression.

**Note:** Examples below show `release-0.22` and `submariner-bundle-0-22`. Substitute your actual target branch and tekton filenames.

Before:
```yaml
pipelinesascode.tekton.dev/on-cel-expression: event == "pull_request" && target_branch
  == "release-0.22"
```

After (substitute your branch and filenames):
```yaml
pipelinesascode.tekton.dev/on-cel-expression: event == "pull_request" && target_branch
  == "release-0.22" && (".tekton/submariner-bundle-0-22-pull-request.yaml".pathChanged()
  || ".tekton/submariner-bundle-0-22-push.yaml".pathChanged() || "bundle.Dockerfile.konflux".pathChanged()
  || "bundle/***".pathChanged() || "config/bundle/***".pathChanged())
```

Do the same for the push file (change `event == "pull_request"` to `event == "push"`).

Verify:
```bash
yq eval '.' .tekton/*.yaml > /dev/null
```

Commit:
```bash
git add .tekton/submariner-bundle-*.yaml
git commit -s -m "Avoid building bundle when updating operator"
```

**Note:** Bundle filters are different from operator filters - they watch `bundle/`, `config/bundle/`, and `bundle.Dockerfile.konflux`.

##### 10. Update Task References

Update Tekton task references to latest versions. This addresses CI warnings about outdated task definitions.

```bash
PATCHER_SHA="b001763bb1cd0286a894cfb570fe12dd7f4504bd"
EXPECTED_SHA256="080ad5d7cf7d0cee732a774b7e4dda0e2ccf26b58e08a8516a3b812bc73beb53"

SCRIPT=$(curl -sL "https://raw.githubusercontent.com/simonbaird/konflux-pipeline-patcher/${PATCHER_SHA}/pipeline-patcher")
ACTUAL_SHA256=$(echo "$SCRIPT" | sha256sum | cut -d' ' -f1)

if [[ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]]; then
  echo "ERROR: Script checksum mismatch!"
  exit 1
fi

echo "$SCRIPT" | bash -s bump-task-refs
git add .tekton/submariner-bundle-*.yaml
git commit -s -m "Optimize Tekton configs for enterprise builds"
```

**Reference:**

- `1daae535` - Optimize Tekton configs for enterprise builds

##### 11. Final Verification

After completing all steps:

```bash
# Review all commits
git log origin/<target-branch>..HEAD

# Verify clean working tree
git status

# Verify spec.params has all required parameters
awk '/^spec:/,/^  pipelineSpec:/' .tekton/submariner-bundle-*-pull-request.yaml | grep -E "name: (dockerfile|build-args-file|build-platforms|hermetic|build-source-image)"
# Should show all 5 parameters with values
```

Expected: Bot commit + commits from Steps 3 and 5-10 (Step 4 is verification only; Steps 5 and 6 may not produce commits if already configured), clean working tree.

All spec.params should include:
- `dockerfile: bundle.Dockerfile.konflux`
- `build-args-file: .tekton/konflux.args`
- `build-platforms: [linux/x86_64, linux/ppc64le, linux/s390x, linux/arm64]`
- `hermetic: "true"`
- `build-source-image: "true"`

Review commits and changes before pushing to update the bot's PR.
