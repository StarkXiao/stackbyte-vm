#!/bin/sh
set -eu

app=stackbyte
version=${VERSION:-dev}
dist=dist
targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"

mkdir -p "$dist"
rm -f "$dist"/*.tar.gz "$dist"/*.zip "$dist"/SHA256SUMS

for target in $targets; do
  goos=${target%/*}
  goarch=${target#*/}
  name="${app}_${version}_${goos}_${goarch}"
  stage="$dist/$name"
  mkdir -p "$stage/examples"
  binary="$stage/$app"
  if [ "$goos" = windows ]; then binary="$binary.exe"; fi
  GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$binary" ./cmd/stackbyte
  cp README.md LICENSE "$stage/"
  cp examples/*.sb "$stage/examples/"
  if [ "$goos" = windows ]; then
    (cd "$dist" && zip -qr "$name.zip" "$name")
  else
    tar -C "$dist" -czf "$dist/$name.tar.gz" "$name"
  fi
  rm -rf "$stage"
done

(cd "$dist" && shasum -a 256 ./*.tar.gz ./*.zip > SHA256SUMS)
echo "Packages written to $dist"
