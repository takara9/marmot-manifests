#!/usr/bin/env bash
# 使い方: ./test.sh http://<LB>/api [回数]
# Cookie を保持して連続アクセスし、振り分け先とカウンターを表示する(jq 必須)
URL="${1:-http://localhost:3000/api}"
N="${2:-10}"
JAR="$(mktemp)"
trap 'rm -f "$JAR"' EXIT

printf "%-4s %-30s %-6s %s\n" "#" "instance" "count" "status"
for i in $(seq 1 "$N"); do
  curl -s -c "$JAR" -b "$JAR" "$URL" |
    jq -r --arg i "$i" '[$i, .instanceId, (.count|tostring),
      (if .persistenceBroken then "NG(switched)" elif .newSession then "new" else "OK" end)] | @tsv' |
    awk -F'\t' '{printf "%-4s %-30s %-6s %s\n", $1, $2, $3, $4}'
done
