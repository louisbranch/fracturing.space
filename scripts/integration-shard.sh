#!/usr/bin/env bash
set -euo pipefail

pkg="${INTEGRATION_SHARD_PACKAGE:-./internal/test/integration}"
tags="${INTEGRATION_SHARD_TAGS:-integration}"

list_top_level_tests() {
	go test -tags="${tags}" "${pkg}" -list '^Test' \
		| awk '/^Test[[:alnum:]_]+$/ { print $1 }' \
		| sort -u
}

shard_for_test() {
	local test_name="$1"
	local shard_total="$2"
	local hash
	hash="$(printf '%s' "${test_name}" | cksum | awk '{print $1}')"
	echo $((hash % shard_total))
}

run_shard() {
	local shard_total shard_index
	shard_total="${INTEGRATION_SHARD_TOTAL:?set INTEGRATION_SHARD_TOTAL}"
	shard_index="${INTEGRATION_SHARD_INDEX:?set INTEGRATION_SHARD_INDEX}"
	if ! [[ "${shard_total}" =~ ^[0-9]+$ ]] || (( shard_total <= 0 )); then
		echo "invalid INTEGRATION_SHARD_TOTAL=${shard_total}" >&2
		exit 1
	fi
	if ! [[ "${shard_index}" =~ ^[0-9]+$ ]] || (( shard_index < 0 )) || (( shard_index >= shard_total )); then
		echo "invalid INTEGRATION_SHARD_INDEX=${shard_index} for total=${shard_total}" >&2
		exit 1
	fi

	mapfile -t all_tests < <(list_top_level_tests)
	if (( ${#all_tests[@]} == 0 )); then
		echo "no top-level integration tests discovered in ${pkg}" >&2
		exit 1
	fi

	local selected=()
	local test_name
	for test_name in "${all_tests[@]}"; do
		if (( "$(shard_for_test "${test_name}" "${shard_total}")" == shard_index )); then
			selected+=("${test_name}")
		fi
	done

	if (( ${#selected[@]} == 0 )); then
		echo "integration shard ${shard_index}/${shard_total} has no assigned tests; skipping"
		exit 0
	fi

	local regex
	regex="^($(IFS='|'; echo "${selected[*]}"))$"
	echo "integration shard ${shard_index}/${shard_total}: ${#selected[@]} tests"
	go test -tags="${tags}" "${pkg}" -run "${regex}" "$@"
}

check_shards() {
	local shard_total
	shard_total="${INTEGRATION_VERIFY_SHARDS_TOTAL:?set INTEGRATION_VERIFY_SHARDS_TOTAL}"
	if ! [[ "${shard_total}" =~ ^[0-9]+$ ]] || (( shard_total <= 0 )); then
		echo "invalid INTEGRATION_VERIFY_SHARDS_TOTAL=${shard_total}" >&2
		exit 1
	fi

	mapfile -t all_tests < <(list_top_level_tests)
	if (( ${#all_tests[@]} == 0 )); then
		echo "no top-level integration tests discovered in ${pkg}" >&2
		exit 1
	fi

	local -a counts
	counts=()
	local i
	for (( i = 0; i < shard_total; i++ )); do
		counts+=("0")
	done

	local test_name shard_index
	for test_name in "${all_tests[@]}"; do
		shard_index="$(shard_for_test "${test_name}" "${shard_total}")"
		if (( shard_index < 0 )) || (( shard_index >= shard_total )); then
			echo "test ${test_name} assigned out-of-range shard ${shard_index}" >&2
			exit 1
		fi
		counts[shard_index]="$((counts[shard_index] + 1))"
	done

	for (( i = 0; i < shard_total; i++ )); do
		echo "integration shard ${i}/${shard_total}: ${counts[i]} tests"
		if (( counts[i] == 0 )); then
			echo "integration shard ${i}/${shard_total} is empty" >&2
			exit 1
		fi
	done
	echo "integration shard coverage check passed for ${#all_tests[@]} tests"
}

mode="run"
if [[ "${1:-}" == "--check" ]]; then
	mode="check"
	shift
fi

if [[ "${mode}" == "check" ]]; then
	check_shards
else
	run_shard "$@"
fi
