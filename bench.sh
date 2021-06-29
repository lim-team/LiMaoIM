#!/bin/bash
readonly messageSize="${1:-200}"
readonly batchSize="${2:-200}"
readonly dataPath="${4:-}"
set -e
set -u

echo "# using  --data-path=$dataPath --size=$messageSize --batch-size=$batchSize"

echo "# compiling/running limao"
pushd cmd/app >/dev/null
go build
./app -e mode="test" >/dev/null 2>&1 &

limao_pid=$!

popd >/dev/null

cleanup() {
    kill -9 $limao_pid
    rm  -rf cmd/app/limaodata-1
}

trap cleanup INT TERM EXIT

echo "# compiling bench_writer/bench_writer"

pushd cmd/bench/bench_writer >/dev/null
go build bench_writer.go
popd >/dev/null

sleep 5

curl -s -o cpu.pprof http://127.0.0.1:1516/debug/pprof/profile &
pprof_pid=$!

echo -n "SEND: "
cmd/bench/bench_writer/bench_writer --size=$messageSize --batch-size=$batchSize 2>&1

echo "waiting for pprof..."
# wait $pprof_pid