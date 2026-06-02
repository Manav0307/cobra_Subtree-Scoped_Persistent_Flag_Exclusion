# Cobra Challenge — Offline Docker Environment

This image provides a **reproducible, fully offline** Debian environment for validating the Subtree-Scoped Persistent Flag Exclusion benchmark after build.

## What is included

| Component | Purpose |
|-----------|---------|
| Debian Bookworm (slim) | Platform-aligned Linux base |
| Go 1.22.10 | Matches CI toolchain floor |
| git, patch | Apply `test.patch` / `solution.patch` |
| bash, gawk | Run `test.sh` and inline JUnit conversion |
| build-essential | Native toolchain if CGO/tests need it |
| ca-certificates | HTTPS during **build only** (Go/module download) |
| `_gotest` user | Root-isolation patterns in benchmark `test.sh` |
| Pre-warmed `GOMODCACHE` / `GOCACHE` | Offline `go test` after image build |

At runtime `GOPROXY=off` and `GOSUMDB=off` prevent any module downloads.

## Build the image

From the repository root (directory containing `Dockerfile`):

```bash
docker build -t cobra-challenge:latest .
```

Optional: override base commit or Go version:

```bash
docker build \
  --build-arg BASE_COMMIT=ad460ea8f249db69c943a365fb84f3a59042d54e \
  --build-arg GO_VERSION=1.22.10 \
  -t cobra-challenge:latest .
```

## Start an interactive container

```bash
docker run --rm -it --network none cobra-challenge:latest
```

`--network none` confirms offline operation (recommended for validation).

Mount local patches if they live outside the image:

```bash
docker run --rm -it --network none \
  -v "$(pwd)/test.patch:/workspace/test.patch:ro" \
  -v "$(pwd)/solution.patch:/workspace/solution.patch:ro" \
  cobra-challenge:latest
```

## Benchmark validation workflow (offline)

Inside the container (already at base commit `ad460ea`):

```bash
# 1. Confirm clean base state
git status
git log -1 --oneline

# 2. Apply challenge patches (order matters)
git apply test.patch
git apply solution.patch

# 3. Run benchmark test modes
bash test.sh base
bash test.sh new

# 4. Full regression (optional)
go test ./...

# 5. JUnit output (when test.sh supports it)
bash test.sh --output_path /tmp/new.xml new
```

Expected benchmark flow **before** `solution.patch`:

```bash
git checkout --force ad460ea8f249db69c943a365fb84f3a59042d54e
git apply test.patch
bash test.sh new    # should FAIL (solution not applied yet)
git apply solution.patch
bash test.sh base   # should PASS
bash test.sh new    # should PASS
```

## One-shot verification (from host)

```bash
docker build -t cobra-challenge:latest .

docker run --rm --network none cobra-challenge:latest \
  bash -lc 'go test -count=1 ./...'

# With patches mounted:
docker run --rm --network none \
  -v "$PWD/test.patch:/workspace/test.patch:ro" \
  -v "$PWD/solution.patch:/workspace/solution.patch:ro" \
  cobra-challenge:latest \
  bash -lc 'git apply test.patch && git apply solution.patch && bash test.sh base && bash test.sh new && go test ./...'
```

## Why this environment is reproducible and offline-safe

1. **Pinned base commit** — `BASE_COMMIT` resets the tree at image build so every container starts from the same baseline.
2. **Pinned Go version** — `GO_VERSION` is fixed at build time, not resolved at runtime.
3. **Pre-downloaded modules** — `go mod download` runs once during build; runtime uses `GOPROXY=off`.
4. **Pre-built test cache** — `go test -count=0 ./...` at base commit fills the build cache before networking is disabled.
5. **No runtime downloads** — `GOSUMDB=off` blocks checksum DB lookups; apt/go/curl are not invoked at runtime.
6. **Root by default** — matches platform benchmark execution; `_gotest` exists for delegated test runs.

## Inspect git history offline

```bash
docker run --rm --network none cobra-challenge:latest \
  bash -lc 'git log --oneline -10 && git show --stat HEAD'
```

## Files

| File | Role |
|------|------|
| `Dockerfile` | Image definition |
| `.dockerignore` | Excludes docs/site noise; keeps build context small |
| `DOCKER.md` | Reviewer instructions (this file) |
