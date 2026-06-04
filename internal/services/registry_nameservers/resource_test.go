// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package registrynameservers_test

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

func TestAccRegistryNameserversResource(t *testing.T) {
	server := newMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConfig(server.URL, "ns1.simply.com", "ns2.simply.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_registry_nameservers.test", "id", "example.com"),
					resource.TestCheckResourceAttr("simply_registry_nameservers.test", "product", "example.com"),
					resource.TestCheckResourceAttr("simply_registry_nameservers.test", "nameservers.0", "ns1.simply.com"),
					resource.TestCheckResourceAttr("simply_registry_nameservers.test", "nameservers.1", "ns2.simply.com"),
				),
			},
			{
				Config: testAccConfig(server.URL, "ns3.simply.com", "ns4.simply.com"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_registry_nameservers.test", "nameservers.0", "ns3.simply.com"),
					resource.TestCheckResourceAttr("simply_registry_nameservers.test", "nameservers.1", "ns4.simply.com"),
				),
			},
			{
				ResourceName:      "simply_registry_nameservers.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	server.assertSetCount(t, 2)
}

func testAccConfig(endpoint string, first string, second string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_registry_nameservers" "test" {
  product = "example.com"
  nameservers = [
    "` + first + `",
    "` + second + `",
  ]
}
`
}

type mockServer struct {
	*httptest.Server
	mu          sync.Mutex
	nameservers []string
	setCount    int
}

func newMockServer(t *testing.T) *mockServer {
	t.Helper()

	mock := &mockServer{
		nameservers: []string{"ns1.simply.com", "ns2.simply.com"},
	}
	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handle))
	return mock
}

func (s *mockServer) handle(w http.ResponseWriter, r *http.Request) {
	if user, pass, ok := r.BasicAuth(); !ok || user != "S123456" || pass != "secret" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if r.URL.Path != "/my/products/example.com/registry/nameservers/" {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		nameservers := append([]string{}, s.nameservers...)
		s.mu.Unlock()
		writeJSON(w, map[string]any{"nameservers": nameservers})
	case http.MethodPut:
		var payload simply.SetNameserversPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			panic(err)
		}
		s.mu.Lock()
		s.nameservers = append([]string{}, payload.Nameservers...)
		s.setCount++
		s.mu.Unlock()
		writeJSON(w, map[string]any{"message": "success"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *mockServer) assertSetCount(t *testing.T, expected int) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.setCount != expected {
		t.Fatalf("expected %d nameserver sets, got %d", expected, s.setCount)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
