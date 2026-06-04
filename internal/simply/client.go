// Copyright Christopher Svensson 2026
// SPDX-License-Identifier: MIT

package simply

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ClientConfig struct {
	AccountName string
	APIKey      string
	Endpoint    string
	Timeout     time.Duration
}

type Client struct {
	accountName string
	apiKey      string
	endpoint    *url.URL
	httpClient  *http.Client
}

type DNSZone struct {
	Name string `json:"name"`
}

type DNSRecord struct {
	RecordID int64   `json:"record_id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Data     string  `json:"data"`
	TTL      int64   `json:"ttl"`
	Priority *int64  `json:"priority"`
	Comment  *string `json:"comment"`
}

type DNSRecordPayload struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Data     string  `json:"data"`
	TTL      int64   `json:"ttl"`
	Priority *int64  `json:"priority,omitempty"`
	Comment  *string `json:"comment,omitempty"`
}

type Product struct {
	Object    string        `json:"object"`
	ObjectURI string        `json:"object_uri"`
	Name      string        `json:"name"`
	Cancelled bool          `json:"cancelled"`
	Domain    ProductDomain `json:"domain"`
	Product   ProductInfo   `json:"product"`
	Servers   ProductServer `json:"servers"`
}

type ProductDomain struct {
	Name      string  `json:"name"`
	NameIDN   string  `json:"name_idn"`
	Managed   bool    `json:"managed"`
	RenewDate *string `json:"date_renewdate"`
}

type ProductInfo struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	CreatedDate string  `json:"date_created"`
	ExpireDate  *string `json:"date_expire"`
}

type ProductServer struct {
	Nameservers []string `json:"nameservers"`
}

type SetNameserversPayload struct {
	Nameservers []string `json:"nameservers"`
}

type DNSSECKey struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type AddDNSSECKeyPayload struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func NewClient(config ClientConfig) *Client {
	endpoint, _ := url.Parse(strings.TrimRight(config.Endpoint, "/"))
	return &Client{
		accountName: config.AccountName,
		apiKey:      config.APIKey,
		endpoint:    endpoint,
		httpClient:  &http.Client{Timeout: config.Timeout},
	}
}

func (c *Client) ListProducts(ctx context.Context) ([]Product, error) {
	var response struct {
		Products []Product `json:"products"`
	}
	err := c.request(ctx, http.MethodGet, "/my/products/", nil, &response)
	return response.Products, err
}

func (c *Client) GetDNSZone(ctx context.Context, product string) (DNSZone, error) {
	var response struct {
		Zone DNSZone `json:"zone"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("/my/products/%s/dns/", url.PathEscape(product)), nil, &response)
	return response.Zone, err
}

func (c *Client) ListDNSRecords(ctx context.Context, product string) ([]DNSRecord, error) {
	var response struct {
		Records []DNSRecord `json:"records"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("/my/products/%s/dns/records/", url.PathEscape(product)), nil, &response)
	return response.Records, err
}

func (c *Client) CreateDNSRecord(ctx context.Context, product string, payload DNSRecordPayload) (int64, error) {
	var response struct {
		Record struct {
			ID int64 `json:"id"`
		} `json:"record"`
	}
	err := c.request(ctx, http.MethodPost, fmt.Sprintf("/my/products/%s/dns/records/", url.PathEscape(product)), payload, &response)
	return response.Record.ID, err
}

func (c *Client) UpdateDNSRecord(ctx context.Context, product string, recordID int64, payload DNSRecordPayload) error {
	return c.request(ctx, http.MethodPut, fmt.Sprintf("/my/products/%s/dns/records/%d/", url.PathEscape(product), recordID), payload, nil)
}

func (c *Client) DeleteDNSRecord(ctx context.Context, product string, recordID int64) error {
	return c.request(ctx, http.MethodDelete, fmt.Sprintf("/my/products/%s/dns/records/%d/", url.PathEscape(product), recordID), nil, nil)
}

func (c *Client) ReloadDNSZone(ctx context.Context, product string) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/my/products/%s/dns/reload/", url.PathEscape(product)), nil, nil)
}

func (c *Client) GetRegistryNameservers(ctx context.Context, product string) ([]string, error) {
	var response struct {
		Nameservers []string `json:"nameservers"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("/my/products/%s/registry/nameservers/", url.PathEscape(product)), nil, &response)
	return response.Nameservers, err
}

func (c *Client) SetRegistryNameservers(ctx context.Context, product string, nameservers []string) error {
	payload := SetNameserversPayload{Nameservers: nameservers}
	return c.request(ctx, http.MethodPut, fmt.Sprintf("/my/products/%s/registry/nameservers/", url.PathEscape(product)), payload, nil)
}

func (c *Client) ListRegistryDNSSECKeys(ctx context.Context, product string) ([]DNSSECKey, error) {
	var response struct {
		Keys []DNSSECKey `json:"dnssec_keys"`
	}
	err := c.request(ctx, http.MethodGet, fmt.Sprintf("/my/products/%s/registry/dnssec/", url.PathEscape(product)), nil, &response)
	return response.Keys, err
}

func (c *Client) AddRegistryDNSSECKey(ctx context.Context, product string, payload AddDNSSECKeyPayload) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/my/products/%s/registry/dnssec/", url.PathEscape(product)), payload, nil)
}

func (c *Client) DeleteRegistryDNSSECKeys(ctx context.Context, product string) error {
	return c.request(ctx, http.MethodDelete, fmt.Sprintf("/my/products/%s/registry/dnssec/", url.PathEscape(product)), nil, nil)
}

func (c *Client) GetDNSRecord(ctx context.Context, product string, recordID int64) (DNSRecord, bool, error) {
	records, err := c.ListDNSRecords(ctx, product)
	if err != nil {
		return DNSRecord{}, false, err
	}
	for _, record := range records {
		if record.RecordID == recordID {
			return record, true, nil
		}
	}
	return DNSRecord{}, false, nil
}

func (c *Client) request(ctx context.Context, method string, path string, body any, target any) error {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	requestURL := c.endpoint.ResolveReference(&url.URL{Path: strings.TrimRight(c.endpoint.Path, "/") + path})
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(c.accountName, c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %d: %s", method, requestURL.String(), resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if target == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}
