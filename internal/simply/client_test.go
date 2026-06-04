// Copyright Christopher Svensson
// SPDX-License-Identifier: MIT

package simply

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientDNSRecordLifecycle(t *testing.T) {
	var created DNSRecordPayload
	var updated DNSRecordPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/my/products/example.com/dns/records/":
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Fatalf("decode create payload: %s", err)
			}
			writeJSON(w, map[string]any{"record": map[string]any{"id": 123}})
		case r.Method == http.MethodGet && r.URL.Path == "/my/products/example.com/dns/records/":
			priority := int64(10)
			writeJSON(w, map[string]any{
				"records": []DNSRecord{
					{RecordID: 123, Name: "www", Type: "A", Data: "192.0.2.10", TTL: 3600},
					{RecordID: 456, Name: "mail", Type: "MX", Data: "mail.example.com", TTL: 3600, Priority: &priority},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/my/products/example.com/dns/records/123/":
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Fatalf("decode update payload: %s", err)
			}
			writeJSON(w, map[string]any{"message": "success"})
		case r.Method == http.MethodDelete && r.URL.Path == "/my/products/example.com/dns/records/123/":
			writeJSON(w, map[string]any{"message": "success"})
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()
	comment := "test record"

	recordID, err := client.CreateDNSRecord(ctx, "example.com", DNSRecordPayload{
		Name:    "www",
		Type:    "A",
		Data:    "192.0.2.10",
		TTL:     3600,
		Comment: &comment,
	})
	if err != nil {
		t.Fatalf("create DNS record: %s", err)
	}
	if recordID != 123 {
		t.Fatalf("expected record ID 123, got %d", recordID)
	}
	if created.Name != "www" || created.Type != "A" || created.Data != "192.0.2.10" || created.TTL != 3600 || created.Comment == nil || *created.Comment != comment {
		t.Fatalf("unexpected create payload: %#v", created)
	}

	record, found, err := client.GetDNSRecord(ctx, "example.com", 456)
	if err != nil {
		t.Fatalf("get DNS record: %s", err)
	}
	if !found {
		t.Fatal("expected DNS record to be found")
	}
	if record.Name != "mail" || record.Type != "MX" || record.Priority == nil || *record.Priority != 10 {
		t.Fatalf("unexpected record: %#v", record)
	}

	err = client.UpdateDNSRecord(ctx, "example.com", 123, DNSRecordPayload{
		Name: "www",
		Type: "A",
		Data: "192.0.2.11",
		TTL:  600,
	})
	if err != nil {
		t.Fatalf("update DNS record: %s", err)
	}
	if updated.Data != "192.0.2.11" || updated.TTL != 600 {
		t.Fatalf("unexpected update payload: %#v", updated)
	}

	if err := client.DeleteDNSRecord(ctx, "example.com", 123); err != nil {
		t.Fatalf("delete DNS record: %s", err)
	}
}

func TestClientDNSZoneAndReload(t *testing.T) {
	reloaded := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/my/products/example.com/dns/":
			writeJSON(w, map[string]any{"zone": map[string]any{"name": "example.com"}})
		case r.Method == http.MethodPost && r.URL.Path == "/my/products/example.com/dns/reload/":
			reloaded = true
			writeJSON(w, map[string]any{"message": "success"})
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	zone, err := client.GetDNSZone(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("get DNS zone: %s", err)
	}
	if zone.Name != "example.com" {
		t.Fatalf("expected zone name example.com, got %q", zone.Name)
	}

	if err := client.ReloadDNSZone(context.Background(), "example.com"); err != nil {
		t.Fatalf("reload DNS zone: %s", err)
	}
	if !reloaded {
		t.Fatal("expected reload endpoint to be called")
	}
}

func TestClientProductsAndRegistry(t *testing.T) {
	var nameserversPayload SetNameserversPayload
	var dnssecPayloads []AddDNSSECKeyPayload
	dnssecDeleted := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/my/products/":
			renewDate := "2027-01-21T00:00:00Z"
			expireDate := "2027-01-21T10:00:00Z"
			writeJSON(w, map[string]any{
				"products": []Product{
					{
						Object:    "example.com",
						ObjectURI: "/2/my/products/example.com/",
						Name:      "example.com",
						Domain: ProductDomain{
							Name:      "example.com",
							NameIDN:   "example.com",
							Managed:   true,
							RenewDate: &renewDate,
						},
						Product: ProductInfo{
							ID:          42,
							Name:        "DNS Service",
							CreatedDate: "2025-01-21T10:00:00Z",
							ExpireDate:  &expireDate,
						},
						Servers: ProductServer{
							Nameservers: []string{"ns1.simply.com", "ns2.simply.com"},
						},
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/my/products/example.com/registry/nameservers/":
			writeJSON(w, map[string]any{"nameservers": []string{"ns1.simply.com", "ns2.simply.com"}})
		case r.Method == http.MethodPut && r.URL.Path == "/my/products/example.com/registry/nameservers/":
			if err := json.NewDecoder(r.Body).Decode(&nameserversPayload); err != nil {
				t.Fatalf("decode nameservers payload: %s", err)
			}
			writeJSON(w, map[string]any{"message": "success"})
		case r.Method == http.MethodGet && r.URL.Path == "/my/products/example.com/registry/dnssec/":
			writeJSON(w, map[string]any{
				"dnssec_keys": []DNSSECKey{{Type: "ds", Data: "12345 13 2 abc123"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/my/products/example.com/registry/dnssec/":
			var payload AddDNSSECKeyPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode dnssec payload: %s", err)
			}
			dnssecPayloads = append(dnssecPayloads, payload)
			writeJSON(w, map[string]any{"message": "success"})
		case r.Method == http.MethodDelete && r.URL.Path == "/my/products/example.com/registry/dnssec/":
			dnssecDeleted = true
			writeJSON(w, map[string]any{"message": "success"})
		default:
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	ctx := context.Background()

	products, err := client.ListProducts(ctx)
	if err != nil {
		t.Fatalf("list products: %s", err)
	}
	if len(products) != 1 || products[0].Object != "example.com" || products[0].Product.ID != 42 {
		t.Fatalf("unexpected products: %#v", products)
	}

	nameservers, err := client.GetRegistryNameservers(ctx, "example.com")
	if err != nil {
		t.Fatalf("get nameservers: %s", err)
	}
	if len(nameservers) != 2 || nameservers[0] != "ns1.simply.com" {
		t.Fatalf("unexpected nameservers: %#v", nameservers)
	}
	if err := client.SetRegistryNameservers(ctx, "example.com", []string{"ns3.simply.com", "ns4.simply.com"}); err != nil {
		t.Fatalf("set nameservers: %s", err)
	}
	if len(nameserversPayload.Nameservers) != 2 || nameserversPayload.Nameservers[0] != "ns3.simply.com" {
		t.Fatalf("unexpected nameservers payload: %#v", nameserversPayload)
	}

	keys, err := client.ListRegistryDNSSECKeys(ctx, "example.com")
	if err != nil {
		t.Fatalf("list dnssec keys: %s", err)
	}
	if len(keys) != 1 || keys[0].Type != "ds" {
		t.Fatalf("unexpected dnssec keys: %#v", keys)
	}
	if err := client.AddRegistryDNSSECKey(ctx, "example.com", AddDNSSECKeyPayload{Type: "ds", Data: "12345 13 2 abc123"}); err != nil {
		t.Fatalf("add dnssec key: %s", err)
	}
	if len(dnssecPayloads) != 1 || dnssecPayloads[0].Data != "12345 13 2 abc123" {
		t.Fatalf("unexpected dnssec payloads: %#v", dnssecPayloads)
	}
	if err := client.DeleteRegistryDNSSECKeys(ctx, "example.com"); err != nil {
		t.Fatalf("delete dnssec keys: %s", err)
	}
	if !dnssecDeleted {
		t.Fatal("expected DNSSEC delete endpoint to be called")
	}
}

func TestClientErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(server.URL)
	_, err := client.ListDNSRecords(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error response")
	}
}

func newTestClient(endpoint string) *Client {
	return NewClient(ClientConfig{
		AccountName: "S123456",
		APIKey:      "secret",
		Endpoint:    endpoint,
		Timeout:     10 * time.Second,
	})
}

func assertBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()

	user, pass, ok := r.BasicAuth()
	if !ok {
		t.Fatal("missing basic auth")
	}
	if user != "S123456" || pass != "secret" {
		t.Fatalf("unexpected basic auth: %q / %q", user, pass)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
