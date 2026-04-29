#!/bin/bash
# Copyright (c) 2025 Pegasystems Inc.
# All rights reserved.
#
# Install pact-ruby-standalone for CI environment
# Based on tenant-resources-operator/distribution/go/installPactLibs.sh

echo "start installPactLibs.sh"
projectDir=$1

tag=1.91.0

filename=pact-${tag}-linux-x86_64.tar.gz

# Create build directory if it doesn't exist
mkdir -p "$projectDir/build"
cd "$projectDir/build" || exit 1

curl -LO https://github.com/pact-foundation/pact-ruby-standalone/releases/download/v${tag}/${filename}
tar xzf ${filename}

echo "Current os: $(uname -sm)"
case $(uname -sm) in
  'Linux x86_64' | 'Darwin x86' | 'Darwin x86_64')
    echo "install pact"
    yes | apk --no-cache add ruby-full gcompat libc6-compat
    ruby -v
    rm -rf /usr/local/pact
    tar -C /usr/local -xzf ${filename}
    mv -v /usr/local/pact/bin/* /usr/local/bin
    mv -v /usr/local/pact/lib/* /usr/local/lib
    mv -v /usr/local/pact/* /usr/local
    ls /usr/local/bin
    ;;
  *)
    echo "Skip pact install (macOS development - use: brew install ruby && gem install pact-mock_service)"
    ;;
esac

rm ${filename}

echo "end installPactLibs.sh"