#!/usr/bin/env bash
set -euo pipefail

# Run only the new subtree-scoped persistent flag exclusion tests.
# This is a benchmark artifact and must not execute the full test suite.

go test -run '^(TestShipdExcludePersistentFlagPreventsInheritedFlags|TestShipdClearExcludedPersistentFlagRestoresInheritedFlags|TestShipdExcludePersistentFlagFiltersDescendants|TestShipdExcludePersistentFlagUpdatesAfterParentFlagAdded|TestShipdExcludePersistentFlagIntegration_ExecuteCAndHelp|TestShipdExcludePersistentFlagIntegration_Completion)$' ./...
