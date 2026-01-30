#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "${script_dir}/.." && pwd)
image_tag=${1:-duality-engine:dev}
data_dir="${repo_root}/data"

mkdir -p "${data_dir}"

if [ "$(id -u)" -eq 0 ]; then
	chown -R 65532:65532 "${data_dir}"
else
	sudo chown -R 65532:65532 "${data_dir}"
fi

docker build -t "${image_tag}" "${repo_root}"

exec docker run \
	-p 127.0.0.1:8081:8081 \
	-v "${data_dir}:/data" \
	-e DUALITY_DB_PATH=/data/duality.db \
	-e DUALITY_GRPC_ADDR=127.0.0.1:8080 \
	-e DUALITY_MCP_ALLOWED_HOSTS=localhost \
	"${image_tag}"
