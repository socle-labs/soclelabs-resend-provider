package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const defaultBaseURL = "https://api.resend.com"

type ResendClient struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func NewResendClient(apiKey string) *ResendClient {
	return &ResendClient{
		APIKey:  apiKey,
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ResendClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "terraform-provider-resend")

	tflog.Debug(ctx, "Resend API request", map[string]interface{}{
		"method": method,
		"url":    url,
	})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response body: %w", err)
	}

	tflog.Debug(ctx, "Resend API response", map[string]interface{}{
		"status_code": resp.StatusCode,
	})

	return respBody, resp.StatusCode, nil
}

// ---------------------------------------------------------------------------
// Domain API
// ---------------------------------------------------------------------------

type CreateDomainRequest struct {
	Name             string `json:"name"`
	Region           string `json:"region,omitempty"`
	CustomReturnPath string `json:"custom_return_path,omitempty"`
}

type DomainRecord struct {
	Record   string `json:"record"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	TTL      string `json:"ttl"`
	Status   string `json:"status"`
	Value    string `json:"value"`
	Priority int    `json:"priority,omitempty"`
}

type Domain struct {
	ID               string            `json:"id"`
	Object           string            `json:"object"`
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	CreatedAt        string            `json:"created_at"`
	Region           string            `json:"region"`
	Records          []DomainRecord    `json:"records,omitempty"`
	CustomReturnPath string            `json:"custom_return_path,omitempty"`
	Capabilities     map[string]string `json:"capabilities,omitempty"`
}

type CreateDomainResponse struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	CreatedAt        string            `json:"created_at"`
	Status           string            `json:"status"`
	Records          []DomainRecord    `json:"records"`
	Region           string            `json:"region"`
	CustomReturnPath string            `json:"custom_return_path,omitempty"`
	Capabilities     map[string]string `json:"capabilities,omitempty"`
}

type UpdateDomainRequest struct {
	OpenTracking  *bool  `json:"open_tracking,omitempty"`
	ClickTracking *bool  `json:"click_tracking,omitempty"`
	TLS           string `json:"tls,omitempty"`
}

func (c *ResendClient) CreateDomain(ctx context.Context, req CreateDomainRequest) (*CreateDomainResponse, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/domains", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result CreateDomainResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) GetDomain(ctx context.Context, id string) (*Domain, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/domains/"+id, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Domain
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) UpdateDomain(ctx context.Context, id string, req UpdateDomainRequest) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodPatch, "/domains/"+id, req)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

func (c *ResendClient) DeleteDomain(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/domains/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

func (c *ResendClient) VerifyDomain(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/domains/"+id+"/verify", nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// API Key API
// ---------------------------------------------------------------------------

type CreateAPIKeyRequest struct {
	Name       string `json:"name"`
	Permission string `json:"permission,omitempty"`
	DomainID   string `json:"domain_id,omitempty"`
}

type CreateAPIKeyResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type APIKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type ListAPIKeysResponse struct {
	Data []APIKey `json:"data"`
}

func (c *ResendClient) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/api-keys", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result CreateAPIKeyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) ListAPIKeys(ctx context.Context) (*ListAPIKeysResponse, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/api-keys", nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result ListAPIKeysResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) DeleteAPIKey(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/api-keys/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Webhook API
// ---------------------------------------------------------------------------

type CreateWebhookRequest struct {
	Endpoint string   `json:"endpoint"`
	Events   []string `json:"events"`
}

type CreateWebhookResponse struct {
	Object        string `json:"object"`
	ID            string `json:"id"`
	SigningSecret string `json:"signing_secret"`
}

type Webhook struct {
	Object    string   `json:"object"`
	ID        string   `json:"id"`
	Endpoint  string   `json:"endpoint"`
	Events    []string `json:"events"`
	CreatedAt string   `json:"created_at"`
}

type UpdateWebhookRequest struct {
	Endpoint string   `json:"endpoint,omitempty"`
	Events   []string `json:"events,omitempty"`
}

func (c *ResendClient) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*CreateWebhookResponse, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/webhooks", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result CreateWebhookResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/webhooks/"+id, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Webhook
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) UpdateWebhook(ctx context.Context, id string, req UpdateWebhookRequest) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodPatch, "/webhooks/"+id, req)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

func (c *ResendClient) DeleteWebhook(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/webhooks/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Contact API
// ---------------------------------------------------------------------------

type CreateContactRequest struct {
	Email        string            `json:"email"`
	FirstName    string            `json:"first_name,omitempty"`
	LastName     string            `json:"last_name,omitempty"`
	Unsubscribed bool              `json:"unsubscribed,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}

type CreateContactResponse struct {
	Object string `json:"object"`
	ID     string `json:"id"`
}

type Contact struct {
	Object       string            `json:"object"`
	ID           string            `json:"id"`
	Email        string            `json:"email"`
	FirstName    string            `json:"first_name"`
	LastName     string            `json:"last_name"`
	Unsubscribed bool              `json:"unsubscribed"`
	CreatedAt    string            `json:"created_at"`
	Properties   map[string]string `json:"properties,omitempty"`
}

type UpdateContactRequest struct {
	Email        string            `json:"email,omitempty"`
	FirstName    string            `json:"first_name,omitempty"`
	LastName     string            `json:"last_name,omitempty"`
	Unsubscribed *bool             `json:"unsubscribed,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}

func (c *ResendClient) CreateContact(ctx context.Context, req CreateContactRequest) (*CreateContactResponse, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/contacts", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result CreateContactResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) GetContact(ctx context.Context, id string) (*Contact, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/contacts/"+id, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Contact
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) UpdateContact(ctx context.Context, id string, req UpdateContactRequest) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodPatch, "/contacts/"+id, req)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

func (c *ResendClient) DeleteContact(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/contacts/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Segment API
// ---------------------------------------------------------------------------

type CreateSegmentRequest struct {
	Name string `json:"name"`
}

type Segment struct {
	Object    string `json:"object"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

func (c *ResendClient) CreateSegment(ctx context.Context, req CreateSegmentRequest) (*Segment, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/segments", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Segment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) GetSegment(ctx context.Context, id string) (*Segment, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/segments/"+id, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Segment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) DeleteSegment(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/segments/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Topic API
// ---------------------------------------------------------------------------

type CreateTopicRequest struct {
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	DefaultSubscription string `json:"default_subscription,omitempty"`
	Visibility          string `json:"visibility,omitempty"`
}

type Topic struct {
	Object              string `json:"object"`
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	DefaultSubscription string `json:"default_subscription"`
	Visibility          string `json:"visibility"`
	CreatedAt           string `json:"created_at"`
}

type UpdateTopicRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
}

func (c *ResendClient) CreateTopic(ctx context.Context, req CreateTopicRequest) (*Topic, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/topics", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Topic
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) GetTopic(ctx context.Context, id string) (*Topic, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/topics/"+id, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result Topic
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) UpdateTopic(ctx context.Context, id string, req UpdateTopicRequest) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodPatch, "/topics/"+id, req)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

func (c *ResendClient) DeleteTopic(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/topics/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Contact Property API
// ---------------------------------------------------------------------------

type CreateContactPropertyRequest struct {
	Key           string `json:"key"`
	Type          string `json:"type"`
	Description   string `json:"description,omitempty"`
	FallbackValue string `json:"fallback_value,omitempty"`
}

type ContactProperty struct {
	Object        string `json:"object"`
	ID            string `json:"id"`
	Key           string `json:"key"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	FallbackValue string `json:"fallback_value"`
	CreatedAt     string `json:"created_at"`
}

type UpdateContactPropertyRequest struct {
	Description   string `json:"description,omitempty"`
	FallbackValue string `json:"fallback_value,omitempty"`
}

func (c *ResendClient) CreateContactProperty(ctx context.Context, req CreateContactPropertyRequest) (*ContactProperty, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodPost, "/contact-properties", req)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK && statusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result ContactProperty
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) GetContactProperty(ctx context.Context, id string) (*ContactProperty, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, "/contact-properties/"+id, nil)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusNotFound {
		return nil, nil
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result ContactProperty
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return &result, nil
}

func (c *ResendClient) UpdateContactProperty(ctx context.Context, id string, req UpdateContactPropertyRequest) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodPatch, "/contact-properties/"+id, req)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

func (c *ResendClient) DeleteContactProperty(ctx context.Context, id string) error {
	body, statusCode, err := c.doRequest(ctx, http.MethodDelete, "/contact-properties/"+id, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	return nil
}

// ---------------------------------------------------------------------------
// List endpoints (for data sources)
// ---------------------------------------------------------------------------

type ListResponse[T any] struct {
	Object  string `json:"object"`
	HasMore bool   `json:"has_more"`
	Data    []T    `json:"data"`
}

func listResource[T any](c *ResendClient, ctx context.Context, path string) ([]T, error) {
	body, statusCode, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", statusCode, string(body))
	}
	var result ListResponse[T]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	return result.Data, nil
}

func (c *ResendClient) ListDomains(ctx context.Context) ([]Domain, error) {
	return listResource[Domain](c, ctx, "/domains")
}

func (c *ResendClient) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	return listResource[Webhook](c, ctx, "/webhooks")
}

func (c *ResendClient) ListContacts(ctx context.Context) ([]Contact, error) {
	return listResource[Contact](c, ctx, "/contacts")
}

func (c *ResendClient) ListSegments(ctx context.Context) ([]Segment, error) {
	return listResource[Segment](c, ctx, "/segments")
}

func (c *ResendClient) ListTopics(ctx context.Context) ([]Topic, error) {
	return listResource[Topic](c, ctx, "/topics")
}

func (c *ResendClient) ListContactProperties(ctx context.Context) ([]ContactProperty, error) {
	return listResource[ContactProperty](c, ctx, "/contact-properties")
}
