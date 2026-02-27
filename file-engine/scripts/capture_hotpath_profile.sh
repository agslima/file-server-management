#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/artifacts/pprof}"
BENCH_TIME="${BENCH_TIME:-20s}"

mkdir -p "$OUT_DIR"

pushd "$ROOT_DIR" >/dev/null

go test ./internal/server \
  -run '^$' \
  -bench 'BenchmarkHandle(Download|UploadComplete)$' \
  -benchmem \
  -benchtime "$BENCH_TIME" \
  -cpuprofile "$OUT_DIR/hotpath.cpu.pprof" \
  -memprofile "$OUT_DIR/hotpath.mem.pprof"

{
  echo "# hot-path profile artifacts"
  echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "bench_time=$BENCH_TIME"
  echo "cpu_profile=$OUT_DIR/hotpath.cpu.pprof"
  echo "mem_profile=$OUT_DIR/hotpath.mem.pprof"
  echo "inspect_cpu=go tool pprof -top $OUT_DIR/hotpath.cpu.pprof"
  echo "inspect_mem=go tool pprof -top $OUT_DIR/hotpath.mem.pprof"
} > "$OUT_DIR/README.txt"

popd >/dev/null

echo "profiles written to $OUT_DIR"
