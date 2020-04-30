#!/bin/bash

set -u
set -o pipefail

#
# THIS SCRIPT SHOULD ONLY BE RUN FROM THE DIRECTORY CONTAINING IT
#
#

##
## KEY VARIABLES ARE THESE TWO
##

# PROPOSED_GOROOT definitely works with 1.14.1, not sure if works with earlier/later versions
PROPOSED_GOROOT=/Users/iansmith/.enable/go1.14.1.src
# PROPOSED_PACKAGES should NOT include unsafe, because it's really implemented by the compiler
PROPOSED_PACKAGES=( \
io errors fmt bytes errors unicode/utf16 math/rand encoding/binary bytes \
)

function testPackage () {
  ## $1 is confirmed path to go installation
  ## $2 proposed package name to be verified
  if [[ -d ${1}/src/${2} ]] && [[ -r ${1}/src/${2} ]] ; then
    return 0
  fi

  echo "unable to find ${2} in src directory of ${1}"
  return 1
}

function testGoRoot () {
  ## $1 is the proposed path to go installation
  ## $? is 0 if the given path is a directory and readable, otherwise 1
  #echo howdy ${1}
  if [[ -d ${1} ]] && [[ -r ${1} ]] && [[ -d ${1}/src ]] && [[ -r ${1}/src ]]; then
    # sanity check
    if [[ ! -f ${1}/VERSION ]]; then
      echo cannot find VERSION file in directory ${1}
      return 1
    fi
    ## looks ok
    return 0
  fi
  echo unable to read directory ${1} or ${1}/src
  return 1
}

function makeSymlink() {
  ## $1 is confirmed path to go installation
  ## remaineder of args are confirmed package name to be linked to this directory
  local goroot
  local dir
  local packages

  goroot=${1}
  shift
  packages=($@)

  for pkg in ${packages[*]}; do

    # if there is an existing link, delete it
    if [[ -L ./${pkg} ]]; then
      rm ./${pkg}
      if [[ ! $? -eq 0 ]]; then
        echo unable to delete existing symlink ${pkg}
        return 4
      fi
    fi

    ## remove intervening directories, if needed
    dir=${pkg}
    while [[ `dirname ${dir}` != "." ]]; do
      rmdir `dirname ${dir}` >& /dev/null
      if [[ ! $? -eq 0 ]]; then
        #echo ignoring local directory `dirname ${dir}`
        true
      fi
      dir=`dirname $dir`
    done

    # make new directory, if needed
    dir=${pkg}
    if [[ `dirname ${dir}` != "." ]]; then
      mkdir -p `dirname ${dir}`
      if [[ ! $? -eq 0 ]]; then
        echo unable to create the directory `dirname ${dir}`
        return 3
      fi
    fi

    # make new link
    ln -s ${goroot}/src/${pkg} ./${pkg}
    if [[ ! $? -eq 0 ]]; then
      echo unable to create symbolic link from ${goroot}/src/${pkg} to ${pkg}
      return 5
    fi

  done
}

function main() {
  ## $1 is the proposed path to go installation
  ## $remainder of args are the proposed list of go packages
  ## $? is 0 on success, non-zero for error

  local goRootOK
  local goroot
  local packageOK
  local packages

  testGoRoot ${1}
  goRootOK=$?

  if [[ ! ${goRootOK} -eq 0 ]]; then
    return ${goRootOK}
  fi

  goroot=${1}
  shift
  packages=($@)

  for pkg in ${packages[*]}; do
    testPackage ${goroot} ${pkg}
    packageOK=$?

    if [[ ! ${packageOK} -eq 0 ]]; then
      return ${packageOK}
    fi

  done
  return 0
}


##
## check that everything is in order
##
main ${PROPOSED_GOROOT} "${PROPOSED_PACKAGES[@]}"
if [[ ! $? -eq 0 ]]; then
  exit $?
fi

##
## make links
##
makeSymlink ${PROPOSED_GOROOT} "${PROPOSED_PACKAGES[@]}"
if [[ ! $? -eq 0 ]]; then
  exit $?
fi

exit 0
