// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package products_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stoffus/terraform-provider-simply/internal/acctest"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

func TestAccProductsDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "S123456" || pass != "secret" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/my/products/" || r.Method != http.MethodGet {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		renewDate := "2027-01-21T00:00:00Z"
		expireDate := "2027-01-21T10:00:00Z"
		writeJSON(w, map[string]any{
			"products": []simply.Product{
				{
					Object:    "example.com",
					ObjectURI: "/2/my/products/example.com/",
					Name:      "example.com",
					Domain: simply.ProductDomain{
						Name:      "example.com",
						NameIDN:   "example.com",
						Managed:   true,
						RenewDate: &renewDate,
					},
					Product: simply.ProductInfo{
						ID:          42,
						Name:        "DNS Service",
						CreatedDate: "2025-01-21T10:00:00Z",
						ExpireDate:  &expireDate,
					},
					Servers: simply.ProductServer{
						Nameservers: []string{"ns1.simply.com", "ns2.simply.com"},
					},
				},
			},
		})
	}))
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.simply_products.test", "id", "products"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.#", "1"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.object", "example.com"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.cancelled", "false"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.domain.name", "example.com"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.domain.managed", "true"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.service.id", "42"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.service.name", "DNS Service"),
					resource.TestCheckResourceAttr("data.simply_products.test", "products.0.nameservers.0", "ns1.simply.com"),
				),
			},
		},
	})
}

func testAccConfig(endpoint string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

data "simply_products" "test" {}
`
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
