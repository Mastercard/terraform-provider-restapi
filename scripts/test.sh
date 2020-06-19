#!/bin/bash

cd $(dirname $0)

export GOOS=""
export GOARCH=""

[[ -z "${GOPATH}" ]] && export GOPATH=$HOME/go

cd ../restapi

echo "Running tests..."
if ! go test;then
  echo "Failed testing. Aborting."
  exit 1
fi
