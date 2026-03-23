#!/bin/bash -e
dir=$(dirname "$2")
[ ! -d "$dir" ] && mkdir -p "$dir"
cp "$1" "$2"
[ -d /downloads ] && chmod 777 /downloads
exec /init
