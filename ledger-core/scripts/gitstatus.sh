#! /usr/bin/env bash
# Usage:
# scripts/gitstatus.sh [-v]
#
# Script checks if working directory is clean and exits with 0. Exits with 1 otherwise.

output=$(git status --porcelain)
if [ $? -ne 0 ]; then
    # `git status` returned an error
    echo "`git status` returned an error"
    exit 1
fi

if [ -z "$output" ]; then
  # Working directory is clean
  exit 0
else
  # There are uncommitted changes
  if [ $1 = "-v" ]; then
    git diff
  else
    git status
  fi
  exit 1
fi
