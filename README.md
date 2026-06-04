# Terraform Provider for Simply.com

This provider manages Simply.com resources through the Simply REST API.

Initial coverage focuses on DNS:

- `simply_dns_record` resource
- `simply_dns_zone` data source
- `simply_dns_records` data source
- `simply_dns_record` data source

## Configuration

The provider follows Simply's documented environment variables:

```sh
export SIMPLY_ACCOUNT_NAME="Sxxxxxx"
export SIMPLY_API_KEY="..."
export SIMPLY_HTTP_TIMEOUT="30"
```

Provider configuration can also set these values directly:

```hcl
provider "simply" {
  account_name = var.simply_account_name
  api_key      = var.simply_api_key
}
```

## Development

```sh
go test ./...
go install
```

Acceptance-style resource tests use a local mock Simply API and do not require
real Simply credentials.

## Releasing

Provider releases are built by GitHub Actions with GoReleaser when a semantic
version tag is pushed, for example `v0.1.0`. The release job signs the checksum
file and uploads the archives, checksums, signature, and registry manifest that
the Terraform Registry expects.

Before the first release:

1. Create an RSA GPG signing key for provider releases.
2. Add the public key to the `stoffus` Terraform Registry namespace.
3. Add the private key and passphrase as GitHub Actions secrets named
   `GPG_PRIVATE_KEY` and `PASSPHRASE`.
4. Publish the public GitHub repository from the Terraform Registry UI.

To release:

```sh
git tag v0.1.0
git push origin v0.1.0
```
