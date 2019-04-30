#!/bin/bash
previous_tag=$(git describe --tags --abbrev=0 $1^)
echo "Commits from $previous_tag to $1:"
git log $previous_tag..$1 --oneline --format="%s" --reverse|sort -s -k1.1,1.4
