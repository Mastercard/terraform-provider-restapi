#!/bin/bash -e

#Build the docs first
cd $(dirname $0)
WORK_DIR=$(pwd)
cd ../

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
cd "$WORK_DIR"

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

if [[ -z "$GPG_PRIVATE_KEY" || -z "$PASSPHRASE" ]];then
  echo "ERROR: GPG_PRIVATE_KEY or PASSPHRASE env variable is not set"
  exit 1
fi

if [[ "$tag" != v* ]];then
  tag="v$tag"
fi

version=$(echo $tag | cut -c 2-)

./test.sh

#Build for all architectures we want
ARTIFACTS=()

echo "Building..."
for GOOS in "${OSs[@]}";do
  for GOARCH in "${ARCHs[@]}";do
    export GOOS GOARCH

    if [[ "$GOOS" = "windows" ]];then
      EXTENSION=".exe"
    else
      EXTENSION=""
    fi
    ZIP_FILE="terraform-provider-restapi_${version}_${GOOS}_${GOARCH}.zip"
    TF_OUT_FILE="terraform-provider-restapi_${tag}${EXTENSION}"
    echo "  $GOOS - $GOARCH: $TF_OUT_FILE"
    go build -o "$TF_OUT_FILE" ../
    zip "$ZIP_FILE" "$TF_OUT_FILE"
    rm -f "$TF_OUT_FILE"
    ARTIFACTS+=("$ZIP_FILE")

    FS_OUT_FILE="fakeserver_$tag-$GOOS-$GOARCH"
    echo "  $FS_OUT_FILE"
    go build -o "$FS_OUT_FILE" ../fakeservercli
    ARTIFACTS+=("$FS_OUT_FILE")
  done
done

shasum -a 256 *.zip > "terraform-provider-restapi_${version}_SHA256SUMS"
ARTIFACTS+=("terraform-provider-restapi_${version}_SHA256SUMS")

#### Signing
export GNUPGHOME=tmpgpg
rm -rf tmpgpg
mkdir tmpgpg
echo "$GPG_PRIVATE_KEY" | gpg --batch --import

gpg --pinentry-mode=loopback --passphrase "$PASSPHRASE" --detach-sign "terraform-provider-restapi_${version}_SHA256SUMS"
ARTIFACTS+=("terraform-provider-restapi_${version}_SHA256SUMS")

#Create the release so we can add our files
./create-github-release.sh github_api_token=$github_api_token owner=$owner repo=$repo tag=$tag draft=false

#Upload all of the files to the release
for FILE in "${ARTIFACTS[@]}";do
  ./upload-github-release-asset.sh github_api_token=$github_api_token owner=$owner repo=$repo tag=$tag filename="$FILE"
done

echo "Cleaning up..."
rm -f release_info.md terraform-provider-restapi_*.zip fakeserver-* *SHA256SUMS*
