package mijnhost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type APIRecord struct {
    Type  string `json:"type"`
    Name  string `json:"name"`
    Value string `json:"value"`
    TTL   int    `json:"ttl"`
}

// client is a mijn.host API client that holds connection info.
type client struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

// newClient creates a client with sensible defaults. It is unexported
// so only code inside the mijnhost package can instantiate it.
func newClient(apiKey string) *client {
    return &client{
        apiKey:     apiKey,
        baseURL:    "https://mijn.host/api/v2",
        httpClient: &http.Client{},
    }
}

// FetchDNS returns the DNS records for a domain.
func (c *client) FetchDNS(domain string) ([]APIRecord, error) {
    req, err := http.NewRequest("GET", c.baseURL+"/domains/"+domain+"/dns", nil)
    if err != nil {
        return nil, err
    }
    req.Header.Add("API-Key", c.apiKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("mijn.host failed to fetch records: %s", resp.Status)
    }

    var apiResponse struct {
        Data struct {
            Records []APIRecord `json:"records"`
        } `json:"data"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
        return nil, err
    }

    return apiResponse.Data.Records, nil
}

// UpdateDNS replaces the DNS records for a domain.
func (c *client) UpdateDNS(domain string, records []map[string]interface{}) error {
    payload := map[string]interface{}{
        "records": records,
    }
    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }
    fmt.Printf("Request body: %s\n", string(body))

    req, err := http.NewRequest("PUT", c.baseURL+"/domains/"+domain+"/dns", bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Add("API-Key", c.apiKey)
    req.Header.Add("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("failed to update records: %s", resp.Status)
    }

    return nil
}

