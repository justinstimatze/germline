#!/usr/bin/env bash
# Every backticked relative path ending in .md that any tracked markdown file
# mentions must exist, resolved against the mentioning file's directory or the
# repo root.
#
# Contract: a backticked relative .md path is a claim that the file exists in
# this repo. Anything absolute or home-relative is somewhere else and is
# skipped — though prefer a URL for another repo, since a path on one machine
# is no use to a reader on another. A path that deliberately does not exist is
# written without backticks.
#
# No pipeline-into-while here on purpose: the first version set a flag inside
# a subshell and reported a clean tree as a failure.
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"
missing=0
for f in $(git ls-files '*.md'); do
  dir=$(dirname "$f")
  for p in $(grep -o '`[A-Za-z0-9_./-]*\.md`' "$f" 2>/dev/null | tr -d '`' | sort -u); do
    case "$p" in /*|~*) continue ;; esac
    if [ ! -e "$dir/$p" ] && [ ! -e "$p" ]; then
      echo "$f: references $p, which does not exist"
      missing=$((missing + 1))
    fi
  done
done
[ "$missing" -eq 0 ]
