#!/usr/bin/env bash
# build-image.sh [TAG] — Cloud Build the gfd (mud + server-go) image from tracked files.
set -euo pipefail
SRC="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${1:-$(git -C "$SRC" rev-parse --short HEAD)}"
PROJECT="${PROJECT:-project-d24a71e9-2daf-4b2d-917}"
CTX="$(mktemp -d)"; trap 'rm -rf "$CTX"' EXIT
git -C "$SRC" ls-files -z -- . ':!docs2' ':!apps2/battlegrounds_gui' | (cd "$SRC" && xargs -0 tar cf -) | tar xf - -C "$CTX"
cp "$SRC/ops/docker/gfd.Dockerfile" "$CTX/Dockerfile"
# CI SA constraints (EMILY/gitops/CI_SETUP.md): pin the staging dir so gcloud does not list buckets
# project-wide (403), and --async + poll instead of streaming logs (needs a Viewer role the CI SA lacks).
BUILD_ID=$(gcloud builds submit "$CTX" --project "$PROJECT" \
  --tag "us-central1-docker.pkg.dev/$PROJECT/emily/gfd:$TAG" \
  --gcs-source-staging-dir="gs://${PROJECT}_cloudbuild/source" \
  --async --format="value(id)")
echo "submitted build $BUILD_ID, polling for completion..."
while true; do
  STATUS=$(gcloud builds describe "$BUILD_ID" --project "$PROJECT" --format="value(status)")
  case "$STATUS" in
    SUCCESS) break ;;
    FAILURE|INTERNAL_ERROR|TIMEOUT|CANCELLED|EXPIRED) echo "build $BUILD_ID: $STATUS" >&2; exit 1 ;;
    *) sleep 5 ;;
  esac
done
echo "gfd:$TAG"
