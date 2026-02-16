# Contributing

Thanks for taking the time to contribute to go-achat-node.

This document describes the preferred workflow and coding standards for this
repository. It is adapted from the contribution guide used in
`cc14514/go-achat-node`.

## Workflow

1. Fork

   Please submit changes via Pull Requests from a fork.

2. Clone

   ```bash
   git clone https://github.com/<your-account>/go-achat-node
   cd go-achat-node
   ```

3. Create a feature branch

   ```bash
   git checkout -b my-change
   ```

4. Build and test

   ```bash
   # Build the reference node binary
   make build

   # Run tests for both modules
   go test ./...
   (cd app/achat && go test ./...)
   ```

5. Keep up to date

   Add the upstream remote and pull updates regularly to reduce conflicts.

   ```bash
   git remote add upstream https://github.com/cc14514/go-achat-node
   git pull upstream master
   ```

6. Push and open a Pull Request

   ```bash
   git push origin my-change
   ```

   Then open a PR on GitHub. If your PR fixes an issue, include `Fixes <url>` in
   the PR description.

7. Clean up branches

   After a PR is merged:

   ```bash
   git push origin :my-change
   git checkout master
   git pull upstream master
   git branch -d my-change
   ```

## Code Review

- Share the PR URL with reviewers after CI passes.
- Reply to every review comment. If you apply a suggestion, say "Done".
- Keep PRs focused and small where possible.

## Coding Standard

### Go style

- Follow the Go style guide: https://go.dev/wiki/CodeReviewComments
- Run `gofmt` on Go files.

### Tests

- Use Go's standard `testing` package.
- Bug fixes MUST include a regression test.
- Any CLI/RPC/P2P contract or behavior change MUST include automated tests.

### Documentation

- Update docs when changing CLI flags, RPC methods, or protocol behavior.
  See `README.md` and `docs/development-guide.zh-CN.md`.

## License

By contributing, you agree that your contributions will be licensed under the
Apache License 2.0, as described in `LICENSE`.
