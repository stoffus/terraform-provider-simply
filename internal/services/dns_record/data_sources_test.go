// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package dnsrecord_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/christopher/terraform-provider-simply/internal/acctest"
	"github.com/christopher/terraform-provider-simply/internal/simply"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSDataSources(t *testing.T) {
	server := newDNSDataSourceMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSDataSourcesConfig(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.simply_dns_zone.test", "id", "example.com"),
					resource.TestCheckResourceAttr("data.simply_dns_zone.test", "name", "example.com"),
					resource.TestCheckResourceAttr("data.simply_dns_records.test", "records.#", "2"),
					resource.TestCheckResourceAttr("data.simply_dns_record.test", "id", "example.com:456"),
					resource.TestCheckResourceAttr("data.simply_dns_record.test", "record_id", "456"),
					resource.TestCheckResourceAttr("data.simply_dns_record.test", "name", "mail"),
					resource.TestCheckResourceAttr("data.simply_dns_record.test", "type", "MX"),
					resource.TestCheckResourceAttr("data.simply_dns_record.test", "data", "mail.example.com"),
					resource.TestCheckResourceAttr("data.simply_dns_record.test", "priority", "10"),
				),
			},
		},
	})
}

func newDNSDataSourceMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "S123456" || pass != "secret" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		switch {
		case r.URL.Path == "/my/products/example.com/dns/" && r.Method == http.MethodGet:
			writeJSON(w, map[string]any{"zone": map[string]any{"name": "example.com"}})
		case r.URL.Path == "/my/products/example.com/dns/records/" && r.Method == http.MethodGet:
			priority := int64(10)
			writeJSON(w, map[string]any{
				"records": []simply.DNSRecord{
					{
						RecordID: 123,
						Name:     "www",
						Type:     "A",
						Data:     "192.0.2.10",
						TTL:      3600,
					},
					{
						RecordID: 456,
						Name:     "mail",
						Type:     "MX",
						Data:     "mail.example.com",
						TTL:      3600,
						Priority: &priority,
					},
				},
			})
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
}

func testAccDNSDataSourcesConfig(endpoint string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

data "simply_dns_zone" "test" {
  product = "example.com"
}

data "simply_dns_records" "test" {
  product = "example.com"
}

data "simply_dns_record" "test" {
  product = "example.com"
  name    = "mail"
  type    = "MX"
}
`
}
