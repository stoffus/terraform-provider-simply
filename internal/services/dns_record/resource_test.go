// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package dnsrecord_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/christopher/terraform-provider-simply/internal/acctest"
	"github.com/christopher/terraform-provider-simply/internal/simply"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecordResource(t *testing.T) {
	server := newDNSMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordResourceConfig(server.URL, "192.0.2.10", 3600, "initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_dns_record.test", "id", "example.com:123"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "product", "example.com"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "record_id", "123"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "name", "www"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "data", "192.0.2.10"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "comment", "initial"),
				),
			},
			{
				Config: testAccDNSRecordResourceConfig(server.URL, "192.0.2.11", 600, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_dns_record.test", "data", "192.0.2.11"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "ttl", "600"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "comment", "updated"),
				),
			},
			{
				ResourceName:      "simply_dns_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	server.assertDeleted(t)
}

func TestAccDNSRecordResourcePriority(t *testing.T) {
	server := newDNSMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordResourceMXConfig(server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_dns_record.test", "type", "MX"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "priority", "10"),
					resource.TestCheckResourceAttr("simply_dns_record.test", "data", "mail.example.com"),
				),
			},
		},
	})
}

func TestAccDNSRecordResourceRecreatesAfterDisappears(t *testing.T) {
	server := newDNSMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordResourceConfig(server.URL, "192.0.2.10", 3600, "initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_dns_record.test", "id", "example.com:123"),
				),
			},
			{
				PreConfig: func() {
					server.forceDelete()
				},
				Config: testAccDNSRecordResourceConfig(server.URL, "192.0.2.10", 3600, "initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("simply_dns_record.test", "id", "example.com:123"),
				),
			},
		},
	})

	server.assertCreateCount(t, 2)
}

func TestAccDNSRecordResourceValidation(t *testing.T) {
	server := newDNSMockServer(t)
	defer server.Close()

	t.Setenv("SIMPLY_ACCOUNT_NAME", "S123456")
	t.Setenv("SIMPLY_API_KEY", "secret")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDNSRecordResourceInvalidTypeConfig(server.URL),
				ExpectError: regexp.MustCompile("must be an uppercase DNS record type"),
			},
			{
				Config:      testAccDNSRecordResourceInvalidTTLConfig(server.URL),
				ExpectError: regexp.MustCompile("Value must be between 1 and 2147483647"),
			},
			{
				Config:      testAccDNSRecordResourceInvalidPriorityConfig(server.URL),
				ExpectError: regexp.MustCompile("Value must be between 0 and 65535"),
			},
		},
	})
}

func testAccDNSRecordResourceConfig(endpoint string, data string, ttl int, comment string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_dns_record" "test" {
  product = "example.com"
  name    = "www"
  type    = "A"
  data    = "` + data + `"
  ttl     = ` + strconv.Itoa(ttl) + `
  comment = "` + comment + `"
}
`
}

func testAccDNSRecordResourceMXConfig(endpoint string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_dns_record" "test" {
  product  = "example.com"
  name     = "@"
  type     = "MX"
  data     = "mail.example.com"
  ttl      = 3600
  priority = 10
}
`
}

func testAccDNSRecordResourceInvalidTypeConfig(endpoint string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_dns_record" "test" {
  product = "example.com"
  name    = "www"
  type    = "a"
  data    = "192.0.2.10"
}
`
}

func testAccDNSRecordResourceInvalidTTLConfig(endpoint string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_dns_record" "test" {
  product = "example.com"
  name    = "www"
  type    = "A"
  data    = "192.0.2.10"
  ttl     = 0
}
`
}

func testAccDNSRecordResourceInvalidPriorityConfig(endpoint string) string {
	return `
provider "simply" {
  endpoint = "` + endpoint + `"
}

resource "simply_dns_record" "test" {
  product  = "example.com"
  name     = "mail"
  type     = "MX"
  data     = "mail.example.com"
  priority = 70000
}
`
}

type dnsMockServer struct {
	*httptest.Server
	mu          sync.Mutex
	record      simply.DNSRecord
	deleted     bool
	createCount int
}

func newDNSMockServer(t *testing.T) *dnsMockServer {
	t.Helper()

	mock := &dnsMockServer{}
	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handle))
	return mock
}

func (s *dnsMockServer) handle(w http.ResponseWriter, r *http.Request) {
	if user, pass, ok := r.BasicAuth(); !ok || user != "S123456" || pass != "secret" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch {
	case r.URL.Path == "/my/products/example.com/dns/records/" && r.Method == http.MethodGet:
		s.handleList(w)
	case r.URL.Path == "/my/products/example.com/dns/records/" && r.Method == http.MethodPost:
		s.handleCreate(w, r)
	case r.URL.Path == "/my/products/example.com/dns/records/123/" && r.Method == http.MethodPut:
		s.handleUpdate(w, r)
	case r.URL.Path == "/my/products/example.com/dns/records/123/" && r.Method == http.MethodDelete:
		s.handleDelete(w)
	default:
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}
}

func (s *dnsMockServer) handleList(w http.ResponseWriter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := []simply.DNSRecord{}
	if !s.deleted && s.record.RecordID != 0 {
		records = append(records, s.record)
	}
	writeJSON(w, map[string]any{"records": records})
}

func (s *dnsMockServer) handleCreate(w http.ResponseWriter, r *http.Request) {
	payload := decodePayload(r)
	s.mu.Lock()
	s.createCount++
	s.record = simply.DNSRecord{
		RecordID: 123,
		Name:     payload.Name,
		Type:     strings.ToUpper(payload.Type),
		Data:     payload.Data,
		TTL:      payload.TTL,
		Priority: payload.Priority,
		Comment:  payload.Comment,
	}
	s.deleted = false
	s.mu.Unlock()

	writeJSON(w, map[string]any{"record": map[string]any{"id": 123}})
}

func (s *dnsMockServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	payload := decodePayload(r)
	s.mu.Lock()
	s.record = simply.DNSRecord{
		RecordID: 123,
		Name:     payload.Name,
		Type:     strings.ToUpper(payload.Type),
		Data:     payload.Data,
		TTL:      payload.TTL,
		Priority: payload.Priority,
		Comment:  payload.Comment,
	}
	s.deleted = false
	s.mu.Unlock()

	writeJSON(w, map[string]any{"message": "success"})
}

func (s *dnsMockServer) handleDelete(w http.ResponseWriter) {
	s.mu.Lock()
	s.deleted = true
	s.mu.Unlock()

	writeJSON(w, map[string]any{"message": "success"})
}

func (s *dnsMockServer) assertDeleted(t *testing.T) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.deleted {
		t.Fatal("expected DNS record to be deleted")
	}
}

func (s *dnsMockServer) forceDelete() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleted = true
}

func (s *dnsMockServer) assertCreateCount(t *testing.T, expected int) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.createCount != expected {
		t.Fatalf("expected %d creates, got %d", expected, s.createCount)
	}
}

func decodePayload(r *http.Request) simply.DNSRecordPayload {
	var payload simply.DNSRecordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		panic(err)
	}
	return payload
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
