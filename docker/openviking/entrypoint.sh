#!/bin/sh
set -eu

api_key="${FRACTURING_SPACE_OPENVIKING_OPENAI_API_KEY:-}"
if [ -z "$api_key" ]; then
  echo "FRACTURING_SPACE_OPENVIKING_OPENAI_API_KEY must be set" >&2
  exit 1
fi

mkdir -p /app/data

cat > /app/ov.conf <<EOF
{
  "embedding": {
    "dense": {
      "api_base": "https://api.openai.com/v1",
      "api_key": "${api_key}",
      "provider": "openai",
      "dimension": 3072,
      "model": "text-embedding-3-large"
    }
  },
  "vlm": {
    "api_base": "https://api.openai.com/v1",
    "api_key": "${api_key}",
    "provider": "openai",
    "model": "gpt-4o"
  },
  "storage": {
    "workspace": "/app/data",
    "agfs": {
      "backend": "local"
    },
    "vectordb": {
      "backend": "local"
    }
  }
}
EOF

python -u /app/port_forward.py 0.0.0.0 1934 127.0.0.1 1933 &
exec openviking-server
