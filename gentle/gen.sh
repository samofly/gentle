#!/bin/sh
#
# This script generates .go files with embedded data from web/ subdirectory.

set -ue

die() {
  echo $1
  exit 1
}

( go-bindata -version ) || die "Could not find go-bindata. Please, install it with:\n\
  go get github.com/samofly/go-bindata/go-bindata"

go-bindata -ignore '.*~'  -prefix 'web' ./web ./web/js ./web/css \
./web/third_party/bootstrap/css \
./web/third_party/bootstrap/fonts \
./web/third_party/bootstrap/js

gofmt -w bindata.go
