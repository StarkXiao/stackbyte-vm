#!/bin/sh
set -eu

# Keep canonical Go spacing, then compact only mechanically safe blocks so the
# project's strict physical-line budget remains reproducible.
files=$(find . -name '*.go' -not -name '*_test.go' -not -path './dist/*')
tests=$(find . -name '*_test.go' -not -path './dist/*')
gofmt -w $files $tests

find . -name '*.go' -not -name '*_test.go' -not -path './dist/*' -print0 |
  xargs -0 perl -0pi -e 's/^\s*\n//mg'
find . -name '*.go' -not -name '*_test.go' -not -path './dist/*' -print0 |
  xargs -0 perl -0pi -e '1 while s/\n([\t ]*)if ([^\n]+) \{\n\1\treturn ([^\n]+)\n\1\}/\n$1if $2 { return $3 }/g; 1 while s/\n([\t ]*)func ([^\n]+) \{\n\1\treturn ([^\n]+)\n\1\}/\n$1func $2 { return $3 }/g'
find . -name '*.go' -not -name '*_test.go' -not -path './dist/*' -print0 |
  xargs -0 perl -0pi -e '1 while s/\n([\t ]*)if ([^\n]+) \{\n\1\t([^\n{}]+)\n\1\}/\n$1if $2 { $3 }/g'
find . -name '*.go' -not -name '*_test.go' -not -path './dist/*' -print0 |
  xargs -0 perl -0pi -e 's/\n([\t ]*)case ([^:\n]+):\n\1\t/\n$1case $2: /g; s/\n([\t ]*)default:\n\1\t/\n$1default: /g'
