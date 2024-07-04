#! /bin/bash
set -euao pipefail

result=$(grep -oE "\\\${IDO_\S+}" "$1" | sort | uniq || true)
if [ "$result" != "" ]; then
  echo "Following environment variables are missing!"
  echo "$result"
  exit 1
fi