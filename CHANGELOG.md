## 0.3.0 (2026-10-23)

FEATURES:

- Authenticate with the API key as a Bearer token; an account name is no longer required (#13, @tomsommer).

BUG FIXES:

- Resolve product names through the account product list, accepting Unicode and punycode domains, case differences, and trailing dots (#14, @tomsommer).
- Fix double-escaped product names in API request paths and report unknown products clearly (#14, @tomsommer).

DEPRECATIONS:

- `account_name` remains accepted but is ignored and deprecated. `SIMPLY_ACCOUNT_NAME` is no longer read. Existing configurations can remove the account name and continue using the same API key.

MAINTENANCE:

- Update GitHub Actions dependencies.
- Retain the working `copywrite` v0.25.2 pin because v0.25.3 fails to compile during documentation generation.
- Correct acceptance-test expectations for TTL and priority validation messages.

## 0.2.0 (2026-06-04)

FEATURES:

- Add `simply_registry_nameservers` for registry nameserver management.
- Add `simply_registry_dnssec_keys` for managing a product's full DNSSEC key set.
- Add the `simply_products` data source for account product discovery.
- Add generated documentation, examples, and mock-based tests for the new resources and data source.

MAINTENANCE:

- Pin `copywrite` to v0.25.2 for reliable documentation generation.

## 0.1.1 (2026-06-04)

BUG FIXES:

- Fix `simply_dns_record` updates after import by preserving the existing Simply record ID during planning.

## 0.1.0 (2026-06-04)

FEATURES:

- Initial release with provider configuration, the `simply_dns_record` resource, and DNS zone and record data sources.
- Add signed provider releases and Terraform Registry publication support.
