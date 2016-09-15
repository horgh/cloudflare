// Package cloudflare provides a basic wrapper around CloudFlare's API. Right
// now I am mainly interested in the DNS related calls.
package cloudflare

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const endpoint = "https://api.cloudflare.com/client/v4/"

// Client holds the information necessary to interact with the API
type Client struct {
	// Key is the API key
	Key string

	// Email is the email on your account
	Email string

	httpClient *http.Client
}

// Response holds generic portions of an API response
type Response struct {
	Success bool
	Errors  []Error
}

// Error holds a single error from an API response.
type Error struct {
	Code    int
	Message string
}

// ListZoneResponse holds the top level List Zone response.
type ListZoneResponse struct {
	Success bool
	Errors  []Error
	Zones   []Zone `json:"result"`
}

// Zone holds the result part of a List Zone response.
type Zone struct {
	ID   string
	Name string
}

// ListDNSResponse holds the response from listing DNS records.
type ListDNSResponse struct {
	Success bool
	Errors  []Error
	Records []DNSRecord `json:"result"`
}

// DNSRecord holds information about a single DNS record.
type DNSRecord struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Proxiable  bool   `json:"proxiable"`
	Proxied    bool   `json:"proxied"`
	TTL        int    `json:"ttl"`
	Locked     bool   `json:"locked"`
	ZoneID     string `json:"zone_id"`
	ZoneName   string `json:"zone_name"`
	CreatedOn  string `json:"created_on"`
	ModifiedOn string `json:"modified_on"`
}

// NewClient creates an API client struct
func NewClient(key, email string) Client {
	client := &http.Client{}
	client.Timeout = time.Duration(60 * time.Second)

	return Client{
		Key:        key,
		Email:      email,
		httpClient: client,
	}
}

// request makes an API request.
func (c Client) request(method, url string, bodyReader io.Reader) ([]byte,
	error) {
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("Unable to create request: %s", err)
	}

	req.Header.Set("X-Auth-Email", c.Email)
	req.Header.Set("X-Auth-Key", c.Key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Request problem: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	err2 := resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("Unable to read body: %s", err)
	}
	if err2 != nil {
		return nil, fmt.Errorf("Problem closing body: %s", err2)
	}

	return body, nil
}

// ListZones makes an API request to list zones.
//
// Parameters:
// name - domain name
// status - May be blank. If so, it defaults to active.
// page - Which page (pagination)
// perPage - How many per page (max 50, min 5)
// order - name, status, email.
// direction - Ordering of listed zones (asc, desc)
// match - Match all search requirements or any (any, all)
//
// Any string parameter, if blank, will use the default. Any integer parameter
// if negative will use the default.
func (c Client) ListZones(name, status string, page, perPage int,
	order, direction, match string) ([]Zone, error) {
	values := url.Values{}

	if len(name) > 0 {
		values.Add("name", name)
	}

	if len(status) == 0 {
		values.Add("status", "active")
	} else {
		values.Add("status", status)
	}

	if page > 0 {
		values.Add("page", fmt.Sprintf("%d", page))
	}

	if perPage > 0 {
		values.Add("per_page", fmt.Sprintf("%d", perPage))
	}

	if len(order) > 0 {
		values.Add("order", order)
	}

	if len(direction) > 0 {
		values.Add("direction", direction)
	}

	if len(match) > 0 {
		values.Add("match", match)
	}

	url := fmt.Sprintf("%szones?%s", endpoint, values.Encode())

	body, err := c.request("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("API request failure: %s", err)
	}

	var zoneResponse ListZoneResponse
	err = json.Unmarshal(body, &zoneResponse)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding problem: %s", err)
	}

	if !zoneResponse.Success {
		return nil, fmt.Errorf("List zone error: %s",
			errorsToError(zoneResponse.Errors))
	}

	return zoneResponse.Zones, nil
}

// ListDNSRecords makes an API request for DNS records.
//
// Parameters:
// zoneID - Zone identifier (see ListZones())
// recordType - May be "A", etc. Blank for all.
// name - The record name. e.g. "example.com" or "mx.example.com". It may be
//   blank to get all.
// content - DNS record content e.g. 127.0.0.1
// page - Page number (pagination)
// perPage - Number per page (min 5, max 100)
// order - How to order records
// direction - Direction to order records.
// match - Whether to match all requirements (all) or any (any).
//
// If a string is empty we will use the default. If an integer is negative
// we will use the default.
func (c Client) ListDNSRecords(zoneID, recordType, name, content string, page,
	perPage int, order, direction, match string) ([]DNSRecord, error) {
	if len(zoneID) == 0 {
		return nil, fmt.Errorf("You must provide a zone ID. Use ListZones() to find one.")
	}

	values := url.Values{}
	if len(recordType) > 0 {
		values.Set("type", recordType)
	}
	if len(name) > 0 {
		values.Set("name", name)
	}
	if len(content) > 0 {
		values.Set("content", content)
	}
	if page > 0 {
		values.Set("page", fmt.Sprintf("%d", page))
	}
	if perPage > 0 {
		values.Set("per_page", fmt.Sprintf("%d", perPage))
	}
	if len(order) > 0 {
		values.Set("order", order)
	}
	if len(direction) > 0 {
		values.Set("direction", direction)
	}
	if len(match) > 0 {
		values.Set("match", match)
	}

	url := fmt.Sprintf("%szones/%s/dns_records?%s", endpoint,
		url.QueryEscape(zoneID), values.Encode())

	body, err := c.request("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("API request failure: %s", err)
	}

	var dnsResponse ListDNSResponse
	err = json.Unmarshal(body, &dnsResponse)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding problem: %s", err)
	}

	if !dnsResponse.Success {
		return nil, fmt.Errorf("List DNS records error: %s",
			errorsToError(dnsResponse.Errors))
	}

	return dnsResponse.Records, nil
}

// UpdateDNSRecord updates a record.
//
// To use this, you should find the record from ListDNSRecords() and then
// change the field(s) you want, and call this function.
// Note several fields are read only:
// ID
// Proxiable
// Locked
// ZoneID
// ZoneName
// CreatedOn
// ModifiedOn
func (c Client) UpdateDNSRecord(record DNSRecord) error {
	jsonPayload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("Unable to encode to JSON: %s", err)
	}

	url := fmt.Sprintf("%szones/%s/dns_records/%s", endpoint,
		url.QueryEscape(record.ZoneID), url.QueryEscape(record.ID))

	bodyReader := bytes.NewReader(jsonPayload)

	body, err := c.request("PUT", url, bodyReader)
	if err != nil {
		return fmt.Errorf("API request failure: %s", err)
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("JSON decoding problem: %s: %s", err, body)
	}

	if !response.Success {
		return fmt.Errorf("Update DNS record error: %s. Payload: %s",
			errorsToError(response.Errors), jsonPayload)
	}

	return nil
}

// We can get back multiple errors from the API. Concatenate them together
// for ease of return.
func errorsToError(apiErrors []Error) error {
	msg := ""

	for _, err := range apiErrors {
		if len(msg) > 0 {
			msg += ", "
		}
		msg += fmt.Sprintf("Code %d: %s", err.Code, err.Message)
	}

	return errors.New(msg)
}
