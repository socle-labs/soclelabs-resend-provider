package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestClient returns a ResendClient pointed at the given httptest.Server.
func newTestClient(server *httptest.Server) *ResendClient {
	c := NewResendClient("re_test_key")
	c.BaseURL = server.URL
	return c
}

// jsonResponse is a helper to write JSON responses in test handlers.
func jsonResponse(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}

// ---------------------------------------------------------------------------
// APIError tests
// ---------------------------------------------------------------------------

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 422, Name: "validation_error", Message: "Missing required field"}
	got := err.Error()
	want := "Resend API error (HTTP 422, validation_error): Missing required field"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAPIError_ErrorWithoutName(t *testing.T) {
	err := &APIError{StatusCode: 500, Message: "Internal server error"}
	got := err.Error()
	want := "Resend API error (HTTP 500): Internal server error"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestIsNotFound(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("expected false for nil error")
	}
	if IsNotFound(&APIError{StatusCode: 422}) {
		t.Error("expected false for 422")
	}
	if !IsNotFound(&APIError{StatusCode: 404}) {
		t.Error("expected true for 404")
	}
}

func TestParseAPIError_StructuredJSON(t *testing.T) {
	body := []byte(`{"statusCode":422,"name":"validation_error","message":"The name field is required"}`)
	err := parseAPIError(422, body)
	if err.StatusCode != 422 {
		t.Errorf("expected status 422, got %d", err.StatusCode)
	}
	if err.Name != "validation_error" {
		t.Errorf("expected name validation_error, got %s", err.Name)
	}
	if err.Message != "The name field is required" {
		t.Errorf("expected message about name field, got %s", err.Message)
	}
}

func TestParseAPIError_RawBody(t *testing.T) {
	body := []byte("something went wrong")
	err := parseAPIError(500, body)
	if err.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", err.StatusCode)
	}
	if err.Message != "something went wrong" {
		t.Errorf("expected raw body as message, got %s", err.Message)
	}
}

func TestCheckError_Allowed(t *testing.T) {
	err := checkError(200, []byte("ok"), 200, 201)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCheckError_NotAllowed(t *testing.T) {
	err := checkError(403, []byte(`{"name":"forbidden","message":"Access denied"}`), 200)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("expected 403, got %d", apiErr.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Domain API tests
// ---------------------------------------------------------------------------

func TestCreateDomain_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/domains" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer re_test_key" {
			t.Error("missing or wrong Authorization header")
		}
		jsonResponse(w, 201, CreateDomainResponse{
			ID:        "dom-123",
			Name:      "example.com",
			Status:    "not_started",
			Region:    "us-east-1",
			CreatedAt: "2024-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateDomain(context.Background(), CreateDomainRequest{
		Name:   "example.com",
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "dom-123" {
		t.Errorf("expected ID dom-123, got %s", resp.ID)
	}
	if resp.Name != "example.com" {
		t.Errorf("expected name example.com, got %s", resp.Name)
	}
}

func TestCreateDomain_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 422, map[string]string{
			"name":    "validation_error",
			"message": "The domain already exists",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.CreateDomain(context.Background(), CreateDomainRequest{Name: "example.com"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("expected 422, got %d", apiErr.StatusCode)
	}
}

func TestGetDomain_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains/dom-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(w, 200, Domain{
			ID:     "dom-123",
			Name:   "example.com",
			Status: "verified",
			Region: "us-east-1",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	domain, err := client.GetDomain(context.Background(), "dom-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain == nil {
		t.Fatal("expected domain, got nil")
	}
	if domain.Status != "verified" {
		t.Errorf("expected status verified, got %s", domain.Status)
	}
}

func TestGetDomain_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 404, map[string]string{"name": "not_found", "message": "Domain not found"})
	}))
	defer server.Close()

	client := newTestClient(server)
	domain, err := client.GetDomain(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain != nil {
		t.Error("expected nil domain for 404")
	}
}

func TestDeleteDomain_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		jsonResponse(w, 200, map[string]interface{}{"object": "domain", "id": "dom-123", "deleted": true})
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.DeleteDomain(context.Background(), "dom-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDomain_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		jsonResponse(w, 200, map[string]interface{}{"id": "dom-123", "object": "domain"})
	}))
	defer server.Close()

	client := newTestClient(server)
	v := true
	err := client.UpdateDomain(context.Background(), "dom-123", UpdateDomainRequest{OpenTracking: &v})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyDomain_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains/dom-123/verify" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(w, 200, map[string]interface{}{"object": "domain", "id": "dom-123"})
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.VerifyDomain(context.Background(), "dom-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// API Key tests
// ---------------------------------------------------------------------------

func TestCreateAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 201, CreateAPIKeyResponse{ID: "key-123", Token: "re_abc123"})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test-key", Permission: "full_access"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "re_abc123" {
		t.Errorf("expected token re_abc123, got %s", resp.Token)
	}
}

func TestListAPIKeys_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ListAPIKeysResponse{
			Data: []APIKey{
				{ID: "key-1", Name: "key-one", CreatedAt: "2024-01-01T00:00:00Z"},
				{ID: "key-2", Name: "key-two", CreatedAt: "2024-01-02T00:00:00Z"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.ListAPIKeys(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 keys, got %d", len(resp.Data))
	}
}

func TestDeleteAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]interface{}{})
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.DeleteAPIKey(context.Background(), "key-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Webhook tests
// ---------------------------------------------------------------------------

func TestCreateWebhook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 201, CreateWebhookResponse{ID: "wh-123", SigningSecret: "whsec_abc"})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateWebhook(context.Background(), CreateWebhookRequest{
		Endpoint: "https://example.com/hook",
		Events:   []string{"email.sent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SigningSecret != "whsec_abc" {
		t.Errorf("expected whsec_abc, got %s", resp.SigningSecret)
	}
}

func TestGetWebhook_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 404, map[string]string{"name": "not_found", "message": "Webhook not found"})
	}))
	defer server.Close()

	client := newTestClient(server)
	webhook, err := client.GetWebhook(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if webhook != nil {
		t.Error("expected nil webhook for 404")
	}
}

func TestUpdateWebhook_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 422, map[string]string{"name": "validation_error", "message": "Invalid endpoint URL"})
	}))
	defer server.Close()

	client := newTestClient(server)
	err := client.UpdateWebhook(context.Background(), "wh-123", UpdateWebhookRequest{Endpoint: "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Name != "validation_error" {
		t.Errorf("expected validation_error, got %s", apiErr.Name)
	}
}

// ---------------------------------------------------------------------------
// Contact tests
// ---------------------------------------------------------------------------

func TestCreateContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, CreateContactResponse{Object: "contact", ID: "ct-123"})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateContact(context.Background(), CreateContactRequest{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "ct-123" {
		t.Errorf("expected ID ct-123, got %s", resp.ID)
	}
}

func TestGetContact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, Contact{
			ID: "ct-123", Email: "test@example.com", FirstName: "Test", LastName: "User",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	contact, err := client.GetContact(context.Background(), "ct-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", contact.Email)
	}
}

func TestGetContact_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := newTestClient(server)
	contact, err := client.GetContact(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact != nil {
		t.Error("expected nil for 404")
	}
}

// ---------------------------------------------------------------------------
// Segment tests
// ---------------------------------------------------------------------------

func TestCreateSegment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, Segment{Object: "segment", ID: "seg-123", Name: "Test Segment"})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateSegment(context.Background(), CreateSegmentRequest{Name: "Test Segment"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "Test Segment" {
		t.Errorf("expected Test Segment, got %s", resp.Name)
	}
}

func TestGetSegment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := newTestClient(server)
	seg, err := client.GetSegment(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seg != nil {
		t.Error("expected nil for 404")
	}
}

// ---------------------------------------------------------------------------
// Topic tests
// ---------------------------------------------------------------------------

func TestCreateTopic_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, Topic{
			Object: "topic", ID: "top-123", Name: "Newsletter",
			DefaultSubscription: "opt_in", Visibility: "public",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateTopic(context.Background(), CreateTopicRequest{Name: "Newsletter"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "top-123" {
		t.Errorf("expected ID top-123, got %s", resp.ID)
	}
}

func TestGetTopic_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := newTestClient(server)
	topic, err := client.GetTopic(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topic != nil {
		t.Error("expected nil for 404")
	}
}

// ---------------------------------------------------------------------------
// Contact Property tests
// ---------------------------------------------------------------------------

func TestCreateContactProperty_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ContactProperty{
			Object: "contact_property", ID: "cp-123", Key: "company_name", Type: "string",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.CreateContactProperty(context.Background(), CreateContactPropertyRequest{
		Key: "company_name", Type: "string",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Key != "company_name" {
		t.Errorf("expected company_name, got %s", resp.Key)
	}
}

func TestGetContactProperty_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := newTestClient(server)
	prop, err := client.GetContactProperty(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prop != nil {
		t.Error("expected nil for 404")
	}
}

// ---------------------------------------------------------------------------
// List endpoint tests
// ---------------------------------------------------------------------------

func TestListDomains_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(w, 200, ListResponse[Domain]{
			Object: "list",
			Data: []Domain{
				{ID: "dom-1", Name: "example.com"},
				{ID: "dom-2", Name: "test.com"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	domains, err := client.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(domains))
	}
}

func TestListDomains_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 401, map[string]string{"name": "unauthorized", "message": "Invalid API key"})
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.ListDomains(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
}

func TestListWebhooks_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ListResponse[Webhook]{
			Data: []Webhook{{ID: "wh-1", Endpoint: "https://a.com"}},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	webhooks, err := client.ListWebhooks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(webhooks))
	}
}

func TestListContacts_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ListResponse[Contact]{Data: []Contact{}})
	}))
	defer server.Close()

	client := newTestClient(server)
	contacts, err := client.ListContacts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(contacts))
	}
}

func TestListSegments_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ListResponse[Segment]{
			Data: []Segment{{ID: "seg-1", Name: "VIP"}},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	segments, err := client.ListSegments(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segments) != 1 {
		t.Errorf("expected 1, got %d", len(segments))
	}
}

func TestListTopics_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ListResponse[Topic]{
			Data: []Topic{{ID: "t-1", Name: "News"}},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	topics, err := client.ListTopics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topics) != 1 {
		t.Errorf("expected 1, got %d", len(topics))
	}
}

func TestListContactProperties_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ListResponse[ContactProperty]{
			Data: []ContactProperty{{ID: "cp-1", Key: "plan", Type: "string"}},
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	props, err := client.ListContactProperties(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Errorf("expected 1, got %d", len(props))
	}
}

// ---------------------------------------------------------------------------
// doRequest edge cases
// ---------------------------------------------------------------------------

func TestDoRequest_ServerDown(t *testing.T) {
	client := NewResendClient("re_test")
	client.BaseURL = "http://127.0.0.1:1" // nothing listening

	_, err := client.CreateDomain(context.Background(), CreateDomainRequest{Name: "example.com"})
	if err == nil {
		t.Fatal("expected error when server is unreachable")
	}
}

func TestDoRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetDomain(context.Background(), "dom-123")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
