#!/bin/sh

major=$1
minor=$2
patch=$3
commit=$4

cd ./cmd/ayd && goversioninfo -platform-specific -ver-major="${major}" -product-ver-major="${major}" -ver-minor="${minor}" -product-ver-minor="${minor}" -ver-patch="${patch}" -product-ver-patch="${patch}" -file-version="${major}.${minor}.${patch} (${commit})" -product-version="${major}.${minor}.${patch} (${commit})"
