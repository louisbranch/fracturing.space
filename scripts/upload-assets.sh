#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'EOF'
Upload image assets to object storage with versioned keys.

Usage:
  scripts/upload-assets.sh \
    --source-dir <dir> \
    --bucket-url <s3://bucket/prefix> \
    --version <v1> \
    --domain <campaign-covers|avatars> \
    --set-id <set_id> \
    --ext <png|webp|jpg|jpeg|avif|gif> \
    [--dry-run]

Example:
  scripts/upload-assets.sh \
    --source-dir ./assets/campaign-covers \
    --bucket-url s3://fracturing-space-assets \
    --version v1 \
    --domain campaign-covers \
    --set-id campaign_cover_set_v1 \
    --ext png
EOF
}

source_dir=""
bucket_url=""
version=""
domain=""
set_id=""
ext=""
dry_run=0
cache_control="public,max-age=31536000,immutable"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--source-dir)
		source_dir="$2"
		shift 2
		;;
	--bucket-url)
		bucket_url="$2"
		shift 2
		;;
	--version)
		version="$2"
		shift 2
		;;
	--domain)
		domain="$2"
		shift 2
		;;
	--set-id)
		set_id="$2"
		shift 2
		;;
	--ext)
		ext="$2"
		shift 2
		;;
	--dry-run)
		dry_run=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		printf 'Unknown argument: %s\n' "$1" >&2
		usage >&2
		exit 1
		;;
	esac
done

if [[ -z "$source_dir" || -z "$bucket_url" || -z "$version" || -z "$domain" || -z "$set_id" || -z "$ext" ]]; then
	printf 'Missing required arguments.\n' >&2
	usage >&2
	exit 1
fi

if ! command -v aws >/dev/null 2>&1; then
	printf 'aws CLI is required (https://docs.aws.amazon.com/cli/).\n' >&2
	exit 1
fi

if [[ ! -d "$source_dir" ]]; then
	printf 'Source directory not found: %s\n' "$source_dir" >&2
	exit 1
fi

lower_ext="$(printf '%s' "$ext" | tr '[:upper:]' '[:lower:]')"
content_type="application/octet-stream"
case "$lower_ext" in
png)
	content_type="image/png"
	;;
webp)
	content_type="image/webp"
	;;
jpg | jpeg)
	content_type="image/jpeg"
	;;
avif)
	content_type="image/avif"
	;;
gif)
	content_type="image/gif"
	;;
esac

shopt -s nullglob
files=("$source_dir"/*."$ext")
if [[ ${#files[@]} -eq 0 ]]; then
	printf 'No files found matching %s/*.%s\n' "$source_dir" "$ext" >&2
	exit 1
fi

bucket_url="${bucket_url%/}"
for file_path in "${files[@]}"; do
	file_name="$(basename "$file_path")"
	asset_id="${file_name%."$ext"}"
	key="$version/$domain/$set_id/$asset_id.$ext"
	destination="$bucket_url/$key"

	if [[ "$dry_run" -eq 1 ]]; then
		printf '[dry-run] aws s3 cp %s %s --content-type %s --cache-control %s\n' \
			"$file_path" "$destination" "$content_type" "$cache_control"
		continue
	fi

	aws s3 cp "$file_path" "$destination" \
		--content-type "$content_type" \
		--cache-control "$cache_control"
done

printf 'Uploaded %d file(s).\n' "${#files[@]}"
