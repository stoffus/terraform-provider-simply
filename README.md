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
