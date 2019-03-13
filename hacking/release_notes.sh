#!/bin/bash
previous_tag=$(git describe --tags --abbrev=0 v$1^)
echo "Commits from $previous_tag to v$1:"
git log $previous_tag..v$1 --oneline --format="%s" --reverse|sort -s -k1.1,1.4
