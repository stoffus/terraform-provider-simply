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
