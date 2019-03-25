#!/usr/bin/env bash
#
# Author: Mastercard
# License: Apache 2
#
# This script accepts the following parameters:
#
# * owner
# * repo
# * tag
# * filename
# * github_api_token
# * draft
#
# Script to create a release GitHub API v3
#
# Example:
#
# create-github-release.sh github_api_token=TOKEN owner=stefanbuck repo=playground tag=v0.1.0 filename=./build.zip draft=true
#

# Check dependencies.
set -e
xargs=$(which gxargs || which xargs)

# Validate settings.
[ "$TRACE" ] && set -x

CONFIG=$@

for line in $CONFIG; do
  eval "$line"
done

# Define variables.
GH_API="https://api.github.com"
GH_REPO="$GH_API/repos/$owner/$repo"
GH_TAGS="$GH_REPO/releases/tags/$tag"
AUTH="Authorization: token $github_api_token"
WGET_ARGS="--content-disposition --auth-no-challenge --no-cookie"
CURL_ARGS="-LJO#"

if [[ "$draft" != 'true' ]]; then
  draft="false"
fi

if [[ "$tag" == 'LATEST' ]]; then
  GH_TAGS="$GH_REPO/releases/latest"
fi

if [[ -z "$github_api_token" || -z "$owner" || -z "$repo" ]];then
  echo "USAGE: $0 github_api_token=TOKEN owner=someone repo=somerepo tag=vX.Y.Z"
  exit 1
fi

# Validate token.
curl -o /dev/null -sH "$AUTH" $GH_REPO || { echo "Error: Invalid repo, token or network issue!";  exit 1; }

if [[ ! -f release_info.md ]];then
  echo "release_info.md file does not exist. Creating it now - hit enter to continue."
  read JUNK
  echo "## New"   > release_info.md
  echo " - "      >> release_info.md
  echo ""         >> release_info.md
  echo "## Fixed" >> release_info.md
  echo " - "      >> release_info.md
  echo ""         >> release_info.md
  vi release_info.md
  if [[ ! -f release_info.md || -z "$(cat release_info.md)" ]];then
    echo "release_info.md file does not exist or is empty. I will not proceed."
    exit 1
  fi
fi

release_info=$(cat release_info.md)
RELEASE_JSON=$(jq -n \
  --arg tag "$tag" \
  --arg name "$tag" \
  --arg body "$release_info" \
  '{ "tag_name": $tag, "name": $name, "body": $body, "draft": '$draft'}' \
  )

# Send the data
response=$(curl -sH "$AUTH" "$GH_REPO/releases" -X POST -d "$RELEASE_JSON")
if [[ $? != 0 ]]; then
  echo "Non-zero response from curl command. Output follows:"
  echo "$response"
  exit 1
fi
