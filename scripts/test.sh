#!/bin/bash -e

cd $(dirname $0)

export PATH="$PATH:$HOME/go/bin"

export GOOS=""
export GOARCH=""

[[ -z "${GOPATH}" ]] && export GOPATH=$HOME/go

cd ../restapi

echo "Running tests..."
if ! go test "$@";then
  echo "Failed testing. Aborting."
  exit 1
fi

#echo "Vetting result..."
#go vet ./...

#echo "Checking for linting..."
#golint -set_exit_status ./...
