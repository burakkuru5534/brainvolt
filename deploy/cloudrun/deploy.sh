#!/usr/bin/env bash
# Deploys the single BrainVolt Cloud Run service (D-096) from
# deploy/cloudrun/api.service.yaml. Runs from Git Bash, Linux or macOS.
#
#   PROJECT_ID=my-project REGION=europe-west1 \
#   IMAGE=europe-west1-docker.pkg.dev/my-project/brainvolt/backend:TAG \
#   FIREBASE_PROJECT_ID=brainvolt POSTGRES_URL_SECRET=brainvolt-postgres-url \
#   ./deploy/cloudrun/deploy.sh
#
# Optional: CONCURRENCY (default 100).
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"
: "${REGION:?set REGION}"
: "${IMAGE:?set IMAGE}"
: "${FIREBASE_PROJECT_ID:?set FIREBASE_PROJECT_ID}"
: "${POSTGRES_URL_SECRET:?set POSTGRES_URL_SECRET}"
CONCURRENCY="${CONCURRENCY:-100}"

dir="$(cd "$(dirname "$0")" && pwd)"
rendered="$(mktemp)"
trap 'rm -f "$rendered"' EXIT

sed -e "s|\${REGION}|${REGION}|g" \
    -e "s|\${IMAGE}|${IMAGE}|g" \
    -e "s|\${CONCURRENCY}|${CONCURRENCY}|g" \
    -e "s|\${FIREBASE_PROJECT_ID}|${FIREBASE_PROJECT_ID}|g" \
    -e "s|\${POSTGRES_URL_SECRET}|${POSTGRES_URL_SECRET}|g" \
    "$dir/api.service.yaml" > "$rendered"

gcloud run services replace "$rendered" --project "$PROJECT_ID" --region "$REGION"

echo
echo "Deployed brainvolt-api (min 0, max 1, concurrency ${CONCURRENCY}, embedded worker)."
echo "If a separate worker service still exists, delete it to stop its charges:"
echo "  gcloud run services delete brainvolt-worker --project $PROJECT_ID --region $REGION"
