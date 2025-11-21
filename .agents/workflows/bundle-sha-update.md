#### Updating Bundle SHAs

**All commands should be run from the repository root directory.**

**Purpose:** Update container image SHAs in the operator bundle to match images built by Konflux. Use this workflow to:
- Refresh SHAs for an existing version (e.g., pull latest 0.21.1 component builds)
- Bump to a new Z-stream version (e.g., 0.21.1 → 0.21.2)

**Before starting, user must:**

```bash
# Be on target release branch
git checkout release-0.21  # Replace 0.21 with your target version

# Login to Konflux cluster
oc login --web https://api.kflux-prd-rh02.0fk9.p1.openshiftapps.com:6443/
```

**Version format guide:**
- **Branch version**: `0.21` (used for git checkout, component queries)
- **Hyphenated version**: `0-21` (used for Konflux component names)
- **Z-stream version**: `v0.21.2` with v prefix (used for make bundle VERSION)
- **Z-stream version**: `0.21.2` without v prefix (used for commit message, PR title)

##### 1. Get Snapshot Information

Find the latest snapshot for your target version that passed tests:

```bash
# Replace 0-21 with your version in hyphenated format (e.g., 0-21 for release-0.21, 0-20 for release-0.20)
# This gets snapshots from both automatic merges (push) and manual retests (retest-comment)
oc get snapshots -n submariner-tenant \
  --sort-by=.metadata.creationTimestamp \
  -o json | jq -r '.items[] |
    select(.metadata.name | startswith("submariner-0-21")) |
    select(.metadata.labels["pac.test.appstudio.openshift.io/event-type"] | IN("push", "retest-comment")) |
    select(.status.conditions[]? | select(.type == "AppStudioTestSucceeded" and .status == "True")) |
    .metadata.name' | tail -n 1
```

**Note:** This query includes both `push` (automatic merge builds) and `retest-comment` (manual retests after infra failures). This ensures you get the latest passing build with actual merged code.

Verify the snapshot details:

```bash
# Use snapshot name from previous command
SNAPSHOT=$(oc get snapshots -n submariner-tenant \
  --sort-by=.metadata.creationTimestamp \
  -o json | jq -r '.items[] |
    select(.metadata.name | startswith("submariner-0-21")) |
    select(.metadata.labels["pac.test.appstudio.openshift.io/event-type"] | IN("push", "retest-comment")) |
    select(.status.conditions[]? | select(.type == "AppStudioTestSucceeded" and .status == "True")) |
    .metadata.name' | tail -n 1)

echo "Latest passing snapshot: $SNAPSHOT"
oc get snapshot $SNAPSHOT -n submariner-tenant \
  -o jsonpath='{.metadata.annotations.test\.appstudio\.openshift\.io/status}' \
  | jq
```

Look for `"status": "TestPassed"` for all scenarios.

**Note on event types:**
- `push`: Automatic build triggered by merge to release branch
- `retest-comment`: Manual retest triggered by comment (e.g., if automatic push build failed due to infra issues)

If you need to manually retrigger a build for a specific branch, comment on the merge commit:
```
/retest branch:release-0.21
```
Replace `release-0.21` with your target branch. This creates a `retest-comment` snapshot that the query above will include.

Extract image SHAs for all components:

```bash
# Get all component images with SHAs
oc get snapshot $SNAPSHOT -n submariner-tenant \
  -o jsonpath='{range .spec.components[*]}{.name}{"\t"}{.containerImage}{"\n"}{end}'
```

**Component name mapping to RELATED_IMAGE variables:**

(Replace `0-21` with your version in hyphenated format)

- `submariner-operator-0-21` → RELATED_IMAGE_submariner-operator
- `submariner-gateway-0-21` → RELATED_IMAGE_submariner-gateway
- `submariner-route-agent-0-21` → RELATED_IMAGE_submariner-routeagent
- `submariner-globalnet-0-21` → RELATED_IMAGE_submariner-globalnet
- `lighthouse-agent-0-21` → RELATED_IMAGE_submariner-lighthouse-agent
- `lighthouse-coredns-0-21` → RELATED_IMAGE_submariner-lighthouse-coredns
- `nettest-0-21` → RELATED_IMAGE_submariner-nettest
- `nettest-0-21` SHA also used for → RELATED_IMAGE_submariner-metrics-proxy

**Note:** The snapshot also includes `subctl-0-21` and `submariner-bundle-0-21` components; these are not used in the bundle's related images and should be ignored.

**Important:** The snapshot shows `quay.io/redhat-user-workloads` URLs, but the config file uses `registry.redhat.io/rhacm2` URLs with different image names (e.g., `submariner-rhel9-operator` instead of `submariner-operator-0-21`). **Only extract and update the SHA256 digest** (the part after `@sha256:`), keeping the existing registry.redhat.io URLs unchanged.

##### 2. Update Related Images File

Edit `config/manager/patches/related-images.deployment.config.yaml` with the new SHAs:

```bash
# Open file for editing
$EDITOR config/manager/patches/related-images.deployment.config.yaml
```

**Update ALL 8 RELATED_IMAGE variables** in the file (using 7 unique SHAs from the snapshot above; nettest SHA is used for both nettest and metrics-proxy).

**Note:** Some component SHAs in the snapshot may already match what's in the config file. This is normal - components build independently, and not all may have changed since the last bundle update. You can update all SHAs for consistency, or only update those that differ.

**Update process for each component:**
1. Find component in snapshot output (e.g., `submariner-operator-0-21`)
2. Copy its SHA256 digest (the part after `@sha256:`)
3. Use mapping above to find corresponding variable (e.g., `RELATED_IMAGE_submariner-operator`)
4. Find that variable in the config file
5. Replace only the SHA256 hash in the `value` field (keep the registry.redhat.io URL)

Example format:
```yaml
- op: add
  path: /spec/template/spec/containers/0/env/-
  value:
    name: RELATED_IMAGE_submariner-operator
    value: registry.redhat.io/rhacm2/submariner-rhel9-operator@sha256:<NEW_SHA>
```

**Note**: The last entry should also update the container image (uses the submariner-operator SHA):
```yaml
- op: replace
  path: /spec/template/spec/containers/0/image
  value: registry.redhat.io/rhacm2/submariner-rhel9-operator@sha256:<NEW_SHA>
```

##### 3. Generate Bundle

Run the bundle generation with the target version:

```bash
make bundle LOCAL_BUILD=1 VERSION=v0.21.2
```

**Choose VERSION based on your scenario:**
- **Version bump**: Use the NEW version you're bumping to (e.g., `VERSION=v0.21.2` when bumping from 0.21.1 to 0.21.2)
- **SHA-only update**: Use the CURRENT version (e.g., `VERSION=v0.21.1` to refresh SHAs without changing bundle version)

To find current version for SHA-only updates:
```bash
grep "^  version:" bundle/manifests/submariner.clusterserviceversion.yaml
# Output example: "  version: 0.21.1"
# Add v prefix when using in make bundle: VERSION=v0.21.1
```

**Note**: VERSION must use semantic versioning with v prefix: vX.Y.Z

**What this does:**
- Generates kustomization files
- Creates/updates bundle manifests in `bundle/manifests/`
- Updates the ClusterServiceVersion (CSV) with new image references
- Validates the bundle structure

**Reference:**

- `d19aa59d` - Update bundle to 0.21.2
- `4ac94990` - Update bundle for 0.21.1

##### 4. Verify Changes

Check that the bundle was updated correctly:

```bash
# Review changed files
git status
```

**Automated SHA verification:**

This script automatically verifies all SHAs in the bundle match the snapshot (critical for correctness):

```bash
# Use snapshot from step 1
SNAPSHOT=submariner-0-21-xxxxx  # Replace with actual snapshot name

echo "=== Verifying SHAs match snapshot $SNAPSHOT ==="
echo ""

# Component to RELATED_IMAGE variable mapping
declare -A COMPONENT_MAP=(
  ["submariner-operator-0-21"]="RELATED_IMAGE_submariner-operator"
  ["submariner-gateway-0-21"]="RELATED_IMAGE_submariner-gateway"
  ["submariner-route-agent-0-21"]="RELATED_IMAGE_submariner-routeagent"
  ["submariner-globalnet-0-21"]="RELATED_IMAGE_submariner-globalnet"
  ["lighthouse-agent-0-21"]="RELATED_IMAGE_submariner-lighthouse-agent"
  ["lighthouse-coredns-0-21"]="RELATED_IMAGE_submariner-lighthouse-coredns"
  ["nettest-0-21"]="RELATED_IMAGE_submariner-nettest"
)

ERRORS=0

# Verify each component SHA matches between snapshot and bundle
for COMPONENT in "${!COMPONENT_MAP[@]}"; do
  VAR_NAME="${COMPONENT_MAP[$COMPONENT]}"

  # Get SHA from snapshot (source of truth)
  SNAPSHOT_SHA=$(oc get snapshot $SNAPSHOT -n submariner-tenant \
    -o jsonpath="{.spec.components[?(@.name=='$COMPONENT')].containerImage}" \
    | grep -o 'sha256:[a-f0-9]*')

  # Get SHA from bundle CSV (what we generated)
  BUNDLE_SHA=$(grep -A1 "name: $VAR_NAME" bundle/manifests/submariner.clusterserviceversion.yaml \
    | grep "value:" | grep -o 'sha256:[a-f0-9]*')

  # Verify both SHAs are present (catch query/grep failures)
  if [ -z "$SNAPSHOT_SHA" ] || [ -z "$BUNDLE_SHA" ]; then
    echo "✗ $COMPONENT: MISSING SHA!"
    echo "  Snapshot: ${SNAPSHOT_SHA:-<empty>}"
    echo "  Bundle:   ${BUNDLE_SHA:-<empty>}"
    ERRORS=$((ERRORS + 1))
  elif [ "$SNAPSHOT_SHA" = "$BUNDLE_SHA" ]; then
    echo "✓ $COMPONENT"
  else
    echo "✗ $COMPONENT: MISMATCH!"
    echo "  Snapshot: $SNAPSHOT_SHA"
    echo "  Bundle:   $BUNDLE_SHA"
    ERRORS=$((ERRORS + 1))
  fi
done

# Verify metrics-proxy uses nettest SHA (special case: same image)
NETTEST_SHA=$(oc get snapshot $SNAPSHOT -n submariner-tenant \
  -o jsonpath="{.spec.components[?(@.name=='nettest-0-21')].containerImage}" \
  | grep -o 'sha256:[a-f0-9]*')
METRICS_SHA=$(grep -A1 "name: RELATED_IMAGE_submariner-metrics-proxy" bundle/manifests/submariner.clusterserviceversion.yaml \
  | grep "value:" | grep -o 'sha256:[a-f0-9]*')

if [ -z "$NETTEST_SHA" ] || [ -z "$METRICS_SHA" ]; then
  echo "✗ metrics-proxy: MISSING SHA!"
  echo "  Expected (nettest): ${NETTEST_SHA:-<empty>}"
  echo "  Bundle:             ${METRICS_SHA:-<empty>}"
  ERRORS=$((ERRORS + 1))
elif [ "$NETTEST_SHA" = "$METRICS_SHA" ]; then
  echo "✓ metrics-proxy (uses nettest SHA)"
else
  echo "✗ metrics-proxy: MISMATCH!"
  echo "  Expected (nettest): $NETTEST_SHA"
  echo "  Bundle:             $METRICS_SHA"
  ERRORS=$((ERRORS + 1))
fi

echo ""
if [ $ERRORS -eq 0 ]; then
  echo "✅ All SHAs verified - bundle matches snapshot!"
else
  echo "❌ VERIFICATION FAILED - $ERRORS mismatches found!"
  echo "DO NOT COMMIT. Review and fix SHA mismatches above."
  exit 1
fi
```

**Note:** This verification catches typos, wrong component mappings, and missed updates. If verification fails, review step 2 and fix the mismatches before committing.

Expected files changed:

**For SHA-only updates** (VERSION matches current version):
- `config/manager/patches/related-images.deployment.config.yaml` (your manual edits)
- `bundle/manifests/submariner.clusterserviceversion.yaml` (generated with new SHAs)

**For version bumps** (VERSION differs from current version):
- `config/manager/patches/related-images.deployment.config.yaml` (your manual edits)
- `bundle/manifests/submariner.clusterserviceversion.yaml` (generated with new SHAs and version)
- `config/manifests/kustomization.yaml` (newTag updated to new version)
- `bundle/metadata/annotations.yaml` (version updated)
- Possibly: `config/bundle/kustomization.yaml`
- Possibly: `bundle/manifests/submariner.io_*.yaml` (CRDs)
- Possibly: `config/crd/bases/submariner.io_*.yaml`
- Possibly: `bundle.Dockerfile`

##### 5. Commit Changes

```bash
# Stage all bundle-related changes
git add config/manager/patches/related-images.deployment.config.yaml
git add bundle/
git add config/manifests/kustomization.yaml config/bundle/kustomization.yaml 2>/dev/null || true

# Commit - use first option for version bumps, second for SHA-only updates
# Option 1: Version bump (replace <z-version> with Z-stream, e.g., 0.21.2)
git commit -s -m "$(cat <<'EOF'
Update bundle to <z-version>

Updates container image SHAs to match Konflux snapshot.
EOF
)"

# Option 2: SHA-only update (no version change)
# git commit -s -m "$(cat <<'EOF'
# Update bundle SHAs to latest
#
# Updates container image SHAs to match Konflux snapshot.
# EOF
# )"
```

Follow commit templates in @.agents/commit-templates.md.

**Reference:**

- `d19aa59d` - Update bundle to 0.21.2

##### 6. Create Pull Request (Optional)

```bash
# Push to your fork (replace with your fork remote name)
git push <your-fork-remote> <current-branch>

# Create PR (replace values in angle brackets)
# For version bump, use "Update bundle to <z-version>" as title
# For SHA-only update, use "Update bundle SHAs to latest" as title
gh pr create \
  --title "Update bundle to <z-version>" \
  --body "Updates bundle image SHAs from Konflux snapshot." \
  --base "release-<y-version>" \
  --assignee "@me"
```

Replace:
- `<your-fork-remote>`: Your fork's git remote name (usually your GitHub username)
- `<current-branch>`: Current branch name
- `<z-version>`: Z-stream version (e.g., `0.21.2`, `0.20.3`) - for version bumps only
- `<y-version>`: Branch version (e.g., `0.21`, `0.20`)

##### Common Issues

- **Validation fails**: Ensure VERSION uses semantic versioning (vX.Y.Z format)
- **Missing LOCAL_BUILD=1**: Without this flag, make will try to run in Dapper container
- **Wrong SHAs**: Double-check you copied the correct SHA for each component from the snapshot
- **Duplicate nettest image**: The Makefile automatically fixes the submariner-nettest name; this is expected
