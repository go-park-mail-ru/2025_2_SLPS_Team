#!/bin/sh
set -e

SRC=/preload/stickers
DST=/shared/stickers

echo "Initializing stickers volume..."

# создаём папку stickers в volume
mkdir -p "$DST"

# копируем ВСЁ содержимое
cp -r "$SRC"/. "$DST"/

echo "Stickers copied to shared volume"

