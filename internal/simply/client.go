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
	"sync"
	"time"
)

type ClientConfig struct {
	APIKey   string
	Endpoint string
	Timeout  time.Duration
}

type Client struct {
	apiKey     string
	endpoint   *url.URL
	httpClient *http.Client

	productsMu     sync.Mutex
	productObjects map[string]string
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
		apiKey:     config.APIKey,
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: config.Timeout},
	}
}

func (c *Client) ListProducts(ctx context.Context) ([]Product, error) {
	var response struct {
		Products []Product `json:"products"`
	}
	err := c.request(ctx, http.MethodGet, "/my/products/", nil, &response)
	return response.Products, err
}

func (c *Client) productObject(ctx context.Context, product string) (string, error) {
	c.productsMu.Lock()
	defer c.productsMu.Unlock()

	if c.productObjects == nil {
		products, err := c.ListProducts(ctx)
		if err != nil {
			return "", fmt.Errorf("list products: %w", err)
		}
		objects := make(map[string]string, len(products)*3)
		for _, p := range products {
			objects[strings.ToLower(p.Object)] = p.Object
		}
		for _, p := range products {
			for _, name := range []string{p.Domain.Name, p.Domain.NameIDN} {
				key := strings.ToLower(name)
				if _, exists := objects[key]; name != "" && !exists {
					objects[key] = p.Object
				}
			}
		}
		c.productObjects = objects
	}

	object, ok := c.productObjects[strings.ToLower(strings.TrimSuffix(product, "."))]
	if !ok {
		return "", fmt.Errorf("product %q not found on the account", product)
	}
	return object, nil
}

func (c *Client) productRequest(ctx context.Context, method string, product string, path string, body any, target any) error {
	object, err := c.productObject(ctx, product)
	if err != nil {
		return err
	}
	return c.request(ctx, method, "/my/products/"+url.PathEscape(object)+path, body, target)
}

func (c *Client) GetDNSZone(ctx context.Context, product string) (DNSZone, error) {
	var response struct {
		Zone DNSZone `json:"zone"`
	}
	err := c.productRequest(ctx, http.MethodGet, product, "/dns/", nil, &response)
	return response.Zone, err
}

func (c *Client) ListDNSRecords(ctx context.Context, product string) ([]DNSRecord, error) {
	var response struct {
		Records []DNSRecord `json:"records"`
	}
	err := c.productRequest(ctx, http.MethodGet, product, "/dns/records/", nil, &response)
	return response.Records, err
}

func (c *Client) CreateDNSRecord(ctx context.Context, product string, payload DNSRecordPayload) (int64, error) {
	var response struct {
		Record struct {
			ID int64 `json:"id"`
		} `json:"record"`
	}
	err := c.productRequest(ctx, http.MethodPost, product, "/dns/records/", payload, &response)
	return response.Record.ID, err
}

func (c *Client) UpdateDNSRecord(ctx context.Context, product string, recordID int64, payload DNSRecordPayload) error {
	return c.productRequest(ctx, http.MethodPut, product, fmt.Sprintf("/dns/records/%d/", recordID), payload, nil)
}

func (c *Client) DeleteDNSRecord(ctx context.Context, product string, recordID int64) error {
	return c.productRequest(ctx, http.MethodDelete, product, fmt.Sprintf("/dns/records/%d/", recordID), nil, nil)
}

func (c *Client) ReloadDNSZone(ctx context.Context, product string) error {
	return c.productRequest(ctx, http.MethodPost, product, "/dns/reload/", nil, nil)
}

func (c *Client) GetRegistryNameservers(ctx context.Context, product string) ([]string, error) {
	var response struct {
		Nameservers []string `json:"nameservers"`
	}
	err := c.productRequest(ctx, http.MethodGet, product, "/registry/nameservers/", nil, &response)
	return response.Nameservers, err
}

func (c *Client) SetRegistryNameservers(ctx context.Context, product string, nameservers []string) error {
	payload := SetNameserversPayload{Nameservers: nameservers}
	return c.productRequest(ctx, http.MethodPut, product, "/registry/nameservers/", payload, nil)
}

func (c *Client) ListRegistryDNSSECKeys(ctx context.Context, product string) ([]DNSSECKey, error) {
	var response struct {
		Keys []DNSSECKey `json:"dnssec_keys"`
	}
	err := c.productRequest(ctx, http.MethodGet, product, "/registry/dnssec/", nil, &response)
	return response.Keys, err
}

func (c *Client) AddRegistryDNSSECKey(ctx context.Context, product string, payload AddDNSSECKeyPayload) error {
	return c.productRequest(ctx, http.MethodPost, product, "/registry/dnssec/", payload, nil)
}

func (c *Client) DeleteRegistryDNSSECKeys(ctx context.Context, product string) error {
	return c.productRequest(ctx, http.MethodDelete, product, "/registry/dnssec/", nil, nil)
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

	requestURL := strings.TrimRight(c.endpoint.String(), "/") + path
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
		return fmt.Errorf("%s %s returned %d: %s", method, requestURL, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if target == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}
