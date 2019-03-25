#!/bin/bash

export GOOS=""
export GOARCH=""

echo "Running tests..."
cd ../restapi
if ! go test;then
  echo "Failed testing. Aborting."
  exit 1
fi
cd -
