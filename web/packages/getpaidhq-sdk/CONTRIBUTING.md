# Contributing to GetPaidHQ SDK

## Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/) for automated versioning and changelog generation.

### Commit Types

- **feat**: A new feature (triggers minor version bump)
- **fix**: A bug fix (triggers patch version bump)
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to CI configuration files and scripts
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

### Breaking Changes

For breaking changes, add `!` after the type or add `BREAKING CHANGE:` in the footer:

```
feat!: remove deprecated API methods

BREAKING CHANGE: The old authentication method has been removed
```

### Examples

```bash
# New feature (minor release)
git commit -m "feat: add subscription cancellation endpoint"

# Bug fix (patch release)
git commit -m "fix: resolve authentication header issue"

# Breaking change (major release)
git commit -m "feat!: redesign API client interface"

# Documentation update (no release)
git commit -m "docs: update README with new examples"
```

## Automated Releases

Releases are fully automated using semantic-release:

1. **Push to main branch** with conventional commits
2. **GitHub Actions** runs tests, lint, and build
3. **semantic-release** determines version bump based on commits
4. **Changelog** is automatically generated
5. **GitHub release** is created
6. **NPM package** is published automatically

### Version Bumping

- `fix:` commits → patch version (1.0.0 → 1.0.1)
- `feat:` commits → minor version (1.0.0 → 1.1.0)
- `BREAKING CHANGE:` → major version (1.0.0 → 2.0.0)

### Pre-release Branches

- **beta branch**: Creates beta pre-releases (1.0.0-beta.1)
- **alpha branch**: Creates alpha pre-releases (1.0.0-alpha.1)

## Development Workflow

1. **Clone and install**:
   ```bash
   git clone <repo>
   cd getpaidhq-sdk
   pnpm install
   ```

2. **Make changes** following conventional commits

3. **Test your changes**:
   ```bash
   pnpm run test
   pnpm run lint
   pnpm run type-check
   pnpm run build
   ```

4. **Commit with conventional format**:
   ```bash
   git commit -m "feat: add new billing endpoint"
   ```

5. **Push to main** - releases happen automatically!

## Required Secrets

For the automated release workflow, ensure these GitHub secrets are set:

- `GITHUB_TOKEN`: Automatically provided by GitHub
- `NPM_TOKEN`: Your NPM access token for publishing

## Local Testing

Test semantic-release locally (dry-run):

```bash
npx semantic-release --dry-run
```