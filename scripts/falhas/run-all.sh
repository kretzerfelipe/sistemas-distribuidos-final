#!/usr/bin/env bash
# Runs every failure scenario in order and prints a summary. Takes a few minutes.
cd "$(dirname "$0")" || exit 1
failed=0
for script in f1-compensation f2-kill f3-duplicate f4-poison f5-race f6-broker f7-late f8-scale; do
  if bash "$script.sh"; then echo "[$script] ok"; else echo "[$script] FAILED"; failed=1; fi
done
exit $failed
