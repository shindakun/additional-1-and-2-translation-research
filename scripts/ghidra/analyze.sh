#!/bin/sh
# Usage: scripts/ghidra/analyze.sh
# Imports both BOOT.BIN files into a Ghidra project in research/ghidra and dumps
# decompiled C, functions and string references to research/ghidra/<game>/.
# Needs analyzeHeadless on PATH, JAVA_HOME on JDK 25+, and ghidra-allegrex installed.
set -eu
root=$(cd "$(dirname "$0")/../.." && pwd)
proj="$root/research/ghidra"
mkdir -p "$proj"
for g in kfaI kfaII; do
	analyzeHeadless "$proj" "kfa/$g" -import "$root/research/extract/$g/PSP_GAME/SYSDIR/BOOT.BIN" \
		-overwrite -scriptPath "$root/scripts/ghidra" \
		-postScript DumpProgram.java "$proj/$g" >"$proj/$g.log" 2>&1 || {
		echo "$g failed, see $proj/$g.log" >&2
		exit 1
	}
	grep -E 'Using (Loader|Language)|REPORT' "$proj/$g.log"
done
