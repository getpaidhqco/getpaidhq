# Changesets

This folder is managed by [changesets](https://github.com/changesets/changesets).

Only `@getpaidhq/sdk` is published to npm. Every other workspace package is either
`private` or listed in the `ignore` array in `config.json`.

## Releasing a new version

1. Make your changes to `packages/getpaidhq-sdk`.
2. Run `pnpm changeset` and describe the change (pick patch/minor/major).
3. Commit the generated file in `.changeset/` along with your code and open a PR.
4. When merged to `main`, the release workflow opens a **Version Packages** PR.
5. Merge that PR — CI builds and publishes `@getpaidhq/sdk` to npm with provenance.
