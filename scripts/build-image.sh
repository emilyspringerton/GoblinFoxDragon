#!/usr/bin/env bash
# build-image.sh [TAG] — Cloud Build the gfd (mud + server-go) image from tracked files.
set -euo pipefail
SRC="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${1:-$(git -C "$SRC" rev-parse --short HEAD)}"
PROJECT="${PROJECT:-project-d24a71e9-2daf-4b2d-917}"
CTX="$(mktemp -d)"; trap 'rm -rf "$CTX"' EXIT
git -C "$SRC" ls-files -z -- . ':!docs2' ':!apps2/battlegrounds_gui' | (cd "$SRC" && xargs -0 tar cf -) | tar xf - -C "$CTX"
cp "$SRC/ops/docker/gfd.Dockerfile" "$CTX/Dockerfile"
gcloud builds submit "$CTX" --project "$PROJECT" --tag "us-central1-docker.pkg.dev/$PROJECT/emily/gfd:$TAG"
echo "gfd:$TAG"
