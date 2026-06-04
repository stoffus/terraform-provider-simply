// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package registrydnssec_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stoffus/terraform-provider-simply/internal/acctest"
	"github.com/stoffus/terraform-provider-simply/internal/simply"
)

func TestAccRegistryDNSSECKeysResource(t *testing.T) {
	server := newMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(server.URL, "12345 13 2 abc123"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_registry_dnssec_keys.test", "id", "example.com"),
					resource.TestCheckResourceAttr("simply_registry_dnssec_keys.test", "product", "example.com"),
					resource.TestCheckResourceAttr("simply_registry_dnssec_keys.test", "keys.0.type", "ds"),
					resource.TestCheckResourceAttr("simply_registry_dnssec_keys.test", "keys.0.data", "12345 13 2 abc123"),
				),
			},
			{
				Config: testAccConfig(server.URL, "67890 13 2 def456"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_registry_dnssec_keys.test", "keys.0.type", "ds"),
					resource.TestCheckResourceAttr("simply_registry_dnssec_keys.test", "keys.0.data", "67890 13 2 def456"),
				),
			},
			{
				ResourceName:      "simply_registry_dnssec_keys.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	server.assertDeleted(t)
	server.assertAddCount(t, 2)
}

func testAccConfig(endpoint string, data string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_registry_dnssec_keys" "test" {
  product = "example.com"
  keys = [
    {
      type = "ds"
      data = "` + data + `"
    },
  ]
}
`
}

type mockServer struct {
	*httptest.Server
	mu          sync.Mutex
	keys        []simply.DNSSECKey
	addCount    int
	deleteCount int
}

func newMockServer(t *testing.T) *mockServer {
	t.Helper()

	mock := &mockServer{}
	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handle))
	return mock
}

func (s *mockServer) handle(w http.ResponseWriter, r *http.Request) {
	if user, pass, ok := r.BasicAuth(); !ok || user != "S123456" || pass != "secret" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if r.URL.Path != "/my/products/example.com/registry/dnssec/" {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		keys := append([]simply.DNSSECKey{}, s.keys...)
		s.mu.Unlock()
		writeJSON(w, map[string]any{"dnssec_keys": keys})
	case http.MethodPost:
		var payload simply.AddDNSSECKeyPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			panic(err)
		}
		s.mu.Lock()
		s.keys = append(s.keys, simply.DNSSECKey{Type: payload.Type, Data: payload.Data})
		s.addCount++
		s.mu.Unlock()
		writeJSON(w, map[string]any{"message": "success"})
	case http.MethodDelete:
		s.mu.Lock()
		s.keys = nil
		s.deleteCount++
		s.mu.Unlock()
		writeJSON(w, map[string]any{"message": "success"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *mockServer) assertDeleted(t *testing.T) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleteCount == 0 {
		t.Fatal("expected DNSSEC delete endpoint to be called")
	}
}

func (s *mockServer) assertAddCount(t *testing.T, expected int) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.addCount != expected {
		t.Fatalf("expected %d DNSSEC key additions, got %d", expected, s.addCount)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
