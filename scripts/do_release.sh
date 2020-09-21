#!/bin/bash -e

#Build the docs first
cd $(dirname $0)/../
tfpluginwebsite
DIFFOUTPUT=`git diff docs`
if [ -n "$DIFFOUTPUT" ];then
  git commit -m 'Update docs before release' docs
  git push
fi

OSs=("darwin" "linux" "windows")
ARCHs=("386" "amd64")

export REST_API_URI="http://127.0.0.1:8082"
[[ -z "${GOPATH}" ]] && export GOPATH=$HOME/go
export CGO_ENABLED=0

#Get into the right directory
cd $(dirname $0)

#Parse command line params
CONFIG=$@
for line in $CONFIG; do
  eval "$line"
done

if [[ -z "$github_api_token" && -f github_api_token ]];then
  github_api_token=$(cat github_api_token)
fi

if [[ -z "$owner" ]];then
  owner="Mastercard"
fi

if [[ -z "$repo" ]];then
  repo="terraform-provider-restapi"
fi

if [[ -z "$github_api_token" || -z "$owner" || -z "$repo" || -z "$tag" ]];then
  echo "USAGE: $0 github_api_token=TOKEN owner=someone repo=somerepo tag=vX.Y.Z"
  exit 1
fi

if [[ "$tag" != v* ]];then
  tag="v$tag"
fi

./test.sh

#Build for all architectures we want
ARTIFACTS=()
#for GOOS in darwin linux windows netbsd openbsd solaris;do

echo "Building..."
for GOOS in "${OSs[@]}";do
  for GOARCH in "${ARCHs[@]}";do
    export GOOS GOARCH

    TF_OUT_FILE="terraform-provider-restapi_$tag-$GOOS-$GOARCH"
    echo "  $TF_OUT_FILE"
    go build -o "$TF_OUT_FILE" ../
    ARTIFACTS+=("$TF_OUT_FILE")

    FS_OUT_FILE="fakeserver-$tag-$GOOS-$GOARCH"
    echo "  $FS_OUT_FILE"
    go build -o "$FS_OUT_FILE" ../fakeservercli
    ARTIFACTS+=("$FS_OUT_FILE")
  done
done

#Create the release so we can add our files
./create-github-release.sh github_api_token=$github_api_token owner=$owner repo=$repo tag=$tag draft=false

#Upload all of the files to the release
for FILE in "${ARTIFACTS[@]}";do
  ./upload-github-release-asset.sh github_api_token=$github_api_token owner=$owner repo=$repo tag=$tag filename="$FILE"
done

echo "Cleaning up..."
rm -f release_info.md
for GOOS in "${OSs[@]}";do
  for GOARCH in "${ARCHs[@]}";do
    rm -f "terraform-provider-restapi_$tag-$GOOS-$GOARCH" "fakeserver-$tag-$GOOS-$GOARCH"
  done
done
