# Contributing

Thanks for working on the Simply.com Terraform provider. Keep changes focused and easy to review.

## Development

Run the test suite before opening or updating a pull request:

```sh
go test ./...
```

Acceptance-style resource tests use a local mock Simply API and do not require real Simply credentials.

## Pull Requests

Open pull requests against `main`. The repository requires review and a passing `test` workflow before merge.

Use a conventional PR title because squash merging uses the PR title as the final commit title. Good examples:

```text
fix: handle missing DNS records
feat: add mail account resource
chore: update provider tooling
```

Prefer small PRs with one purpose. Link the issue being fixed when one exists, describe the implementation approach, and call out any security-control impact in the PR template.

## Releases

Releases are created by pushing a semantic version tag, such as `v0.1.0`. See `README.md` for release setup and tagging details.
