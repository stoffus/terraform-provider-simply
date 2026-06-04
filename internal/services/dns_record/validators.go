// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package dnsrecord

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func productValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthAtLeast(1),
	}
}

func recordNameValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthAtLeast(1),
	}
}

func recordTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Z][A-Z0-9]*$`), "must be an uppercase DNS record type such as A, AAAA, CNAME, MX, TXT, SRV, or CAA"),
	}
}

func recordDataValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthAtLeast(1),
	}
}

func ttlValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.Between(1, 2147483647),
	}
}

func priorityValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.Between(0, 65535),
	}
}
