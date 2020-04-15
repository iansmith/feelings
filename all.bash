#!/bin/bash
set -e

function builddir() {
  echo
  echo -n '**** '
  echo -n $2
  echo ' **** '
  pushd $1 >& /dev/null; make; popd >& /dev/null
}

builddir src/anticipation/cmd/antc 'anticipation bootloader '
builddir src/anticipation/cmd/release  'release: anticipation host side transmitter'
builddir src/joy 'joy: microkernel'
