#!/usr/bin/env bash
set -euo pipefail

image_repo="louisbranch/duality-engine"
local_tag=${1:-duality-engine:dev}
remote_tag=${2:-latest}

full_tag="${image_repo}:${remote_tag}"

docker tag "${local_tag}" "${full_tag}"
docker push "${full_tag}"
