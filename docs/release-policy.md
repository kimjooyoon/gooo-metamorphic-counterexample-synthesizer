# Release policy

Delivery is ordered as feature branch, pull request, GitHub Actions, merge,
post-main GitHub Actions, then annotated release. The release workflow binds
the tag to the exact merged main commit, creates a draft with the standard
`GITHUB_TOKEN`, uploads the source and evidence assets, publishes the release,
and verifies the public release API, annotated tag target, asset sizes, and
SHA-256 digests.

The workflow does not query any immutable-release administration or capability
endpoint. Existing tags and releases are never moved, deleted, or replaced;
the workflow fails if the requested tag already exists.
