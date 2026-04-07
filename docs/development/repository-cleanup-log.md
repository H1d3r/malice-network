# Repository Cleanup Log

## Overview

This document records the repository cleanup work completed in the local repository during the current maintenance session.

The work focused on:

- removing leaked configuration and service-specific references from Git history
- removing `OneDrive` and `SharePoint` related plugin traces
- removing committed binary payload resources from the main repository history
- adding ignore rules for local-only generated artifacts and local community resources

This document is a maintenance record. It does not change the current submodule layout.

## Scope

The cleanup covered the main repository only.

The following were intentionally excluded from cleanup changes:

- `external/IoM-go`
- other `external/` submodules
- the current submodule path definition for the community intl repository

## History Rewrites Performed

The local Git history was rewritten to remove or scrub the following:

- `OneDrive` / `SharePoint` related code and text references
- leaked Azure application secret and associated connection strings
- committed community binary resource directories

The following historical paths were removed from the main repository history:

- `client/command/onedrive`
- `helper/intl/custom/onedrive`
- `helper/intl/community/resources`
- `helper/consts/community/resources`

The following content classes were scrubbed from history:

- `OneDrive`
- `SharePoint`
- Azure application secret values
- connection strings containing the leaked secret

## Current OneDrive Status

Current verification results:

- no `OneDrive` or `SharePoint` matches were found in the current tracked working tree
- no `OneDrive` or `SharePoint` matches were found in local commit messages
- no `OneDrive` or `SharePoint` matches were found in local commit diffs

Searches used during verification included:

- case-insensitive content scan of the repository working tree
- case-insensitive filename and path scan
- `git log --grep` over all local refs
- `git log -G` over all local refs

## Binary Cleanup Status

The main repository history previously contained committed binary payload resources under community resource directories.

These historical directories were removed from the main repository history:

- `helper/intl/community/resources`
- `helper/consts/community/resources`

After the history rewrite, the main repository no longer reports tracked historical blobs with extensions such as:

- `.exe`
- `.dll`
- `.bin`
- `.db`
- `.sqlite`

The following categories were intentionally not treated as repository pollution:

- documentation images under `docs/assets`
- dependency content under `external/`

## Ignore Rules Added

The local `.gitignore` was extended to prevent reintroducing local-only files.

Added ignore coverage includes:

- local config files
- local auth and log files
- local build outputs
- local test executables
- local community resource directories and files

Relevant current ignore entries include:

- `/config.yaml`
- `/server/config.yaml`
- `*.auth`
- `*.log`
- `*.test.exe`
- `/client.exe`
- `/server.exe`
- `/client_linux_amd64`
- `/server_linux_amd64`
- `/core`
- `/testinventory.exe`
- `helper/intl/community/resources/`
- `helper/intl/community/modules/`
- `helper/intl/community/main.lua`
- `helper/intl/community/mal.yaml`

## Local Commits Added

The following local commits were added as part of the cleanup session:

- `394fc2d1` `chore: ignore local community resources and build artifacts`
- `5d09f1bf` `chore: ignore local build outputs`

Earlier local history was also rewritten, so commit hashes before the rewrite should not be treated as stable references.

## Current Known Issues

The submodule path definition for the community intl repository is currently inconsistent with the local initialized working tree layout.

Observed state:

- `.gitmodules` records the submodule path as `helper/intl/community/community`
- the local initialized submodule working tree is rooted at `helper/intl/community`

This inconsistency was detected during the cleanup session but was intentionally left unchanged.

Current working tree items that remain outside the scope of this cleanup:

- modified submodule state in `external/IoM-go`
- unresolved community intl submodule path mismatch

## Verification Summary

Completed verification steps:

1. verified that `OneDrive` / `SharePoint` content no longer appears in the main repository working tree
2. verified that `OneDrive` / `SharePoint` content no longer appears in local Git history searches
3. verified that previously committed community binary resource paths no longer appear in local Git history
4. verified that local ignore rules cover the identified generated artifacts

## Operational Note

Because Git history was rewritten, any remote synchronization must use forced push operations from a network environment that can successfully push to GitHub.

This document records local repository state only. It does not guarantee that the remote repository has already been updated.
