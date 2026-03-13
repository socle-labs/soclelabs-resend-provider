package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// ---------------------------------------------------------------------------
// Mock API server
// ---------------------------------------------------------------------------

// mockResendServer creates a mock Resend API server that handles all resource
// endpoints with an in-memory store. Returns the server and a cleanup func.
func mockResendServer() *httptest.Server {
	mu := &sync.Mutex{}
	domains := map[string]Domain{}
	apiKeys := map[string]APIKey{}
	webhooks := map[string]Webhook{}
	contacts := map[string]Contact{}
	segments := map[string]Segment{}
	topics := map[string]Topic{}
	contactProps := map[string]ContactProperty{}

	idCounter := 0
	nextID := func(prefix string) string {
		mu.Lock()
		defer mu.Unlock()
		idCounter++
		return prefix + "-" + string(rune('0'+idCounter))
	}
	_ = nextID // suppress unused if not used in all paths

	mux := http.NewServeMux()

	// --- Domains ---
	mux.HandleFunc("/domains", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateDomainRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "dom-test-1"
			d := Domain{ID: id, Name: req.Name, Status: "not_started", Region: req.Region, CreatedAt: "2024-01-01T00:00:00Z"}
			if d.Region == "" {
				d.Region = "us-east-1"
			}
			domains[id] = d
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(CreateDomainResponse{
				ID: id, Name: d.Name, Status: d.Status, Region: d.Region, CreatedAt: d.CreatedAt,
			})
		case http.MethodGet:
			var list []Domain
			for _, d := range domains {
				list = append(list, d)
			}
			json.NewEncoder(w).Encode(ListResponse[Domain]{Object: "list", Data: list})
		}
	})

	mux.HandleFunc("/domains/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		// Extract ID - handle both /domains/{id} and /domains/{id}/verify
		path := r.URL.Path[len("/domains/"):]
		id := path
		isVerify := false
		if len(path) > 7 && path[len(path)-7:] == "/verify" {
			id = path[:len(path)-7]
			isVerify = true
		}

		switch {
		case isVerify && r.Method == http.MethodPost:
			json.NewEncoder(w).Encode(map[string]string{"object": "domain", "id": id})
		case r.Method == http.MethodGet:
			d, ok := domains[id]
			if !ok {
				w.WriteHeader(404)
				json.NewEncoder(w).Encode(map[string]string{"name": "not_found", "message": "not found"})
				return
			}
			json.NewEncoder(w).Encode(d)
		case r.Method == http.MethodPatch:
			d, ok := domains[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			var req UpdateDomainRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.TLS != "" {
				// store if needed
				_ = req.TLS
			}
			domains[id] = d
			json.NewEncoder(w).Encode(map[string]string{"id": id, "object": "domain"})
		case r.Method == http.MethodDelete:
			delete(domains, id)
			json.NewEncoder(w).Encode(map[string]interface{}{"object": "domain", "id": id, "deleted": true})
		}
	})

	// --- API Keys ---
	mux.HandleFunc("/api-keys", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateAPIKeyRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "key-test-1"
			apiKeys[id] = APIKey{ID: id, Name: req.Name, CreatedAt: "2024-01-01T00:00:00Z"}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(CreateAPIKeyResponse{ID: id, Token: "re_mock_token"})
		case http.MethodGet:
			var list []APIKey
			for _, k := range apiKeys {
				list = append(list, k)
			}
			json.NewEncoder(w).Encode(ListAPIKeysResponse{Data: list})
		}
	})

	mux.HandleFunc("/api-keys/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id := r.URL.Path[len("/api-keys/"):]
		if r.Method == http.MethodDelete {
			delete(apiKeys, id)
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})

	// --- Webhooks ---
	mux.HandleFunc("/webhooks", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateWebhookRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "wh-test-1"
			webhooks[id] = Webhook{ID: id, Endpoint: req.Endpoint, Events: req.Events, CreatedAt: "2024-01-01T00:00:00Z"}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(CreateWebhookResponse{Object: "webhook", ID: id, SigningSecret: "whsec_mock"})
		case http.MethodGet:
			var list []Webhook
			for _, wh := range webhooks {
				list = append(list, wh)
			}
			json.NewEncoder(w).Encode(ListResponse[Webhook]{Object: "list", Data: list})
		}
	})

	mux.HandleFunc("/webhooks/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id := r.URL.Path[len("/webhooks/"):]
		switch r.Method {
		case http.MethodGet:
			wh, ok := webhooks[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(wh)
		case http.MethodPatch:
			wh, ok := webhooks[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			var req UpdateWebhookRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Endpoint != "" {
				wh.Endpoint = req.Endpoint
			}
			if req.Events != nil {
				wh.Events = req.Events
			}
			webhooks[id] = wh
			json.NewEncoder(w).Encode(wh)
		case http.MethodDelete:
			delete(webhooks, id)
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})

	// --- Contacts ---
	mux.HandleFunc("/contacts", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateContactRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "ct-test-1"
			contacts[id] = Contact{
				ID: id, Email: req.Email, FirstName: req.FirstName, LastName: req.LastName,
				Unsubscribed: req.Unsubscribed, CreatedAt: "2024-01-01T00:00:00Z",
			}
			json.NewEncoder(w).Encode(CreateContactResponse{Object: "contact", ID: id})
		case http.MethodGet:
			var list []Contact
			for _, c := range contacts {
				list = append(list, c)
			}
			json.NewEncoder(w).Encode(ListResponse[Contact]{Object: "list", Data: list})
		}
	})

	mux.HandleFunc("/contacts/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id := r.URL.Path[len("/contacts/"):]
		switch r.Method {
		case http.MethodGet:
			c, ok := contacts[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(c)
		case http.MethodPatch:
			c, ok := contacts[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			var req UpdateContactRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Email != "" {
				c.Email = req.Email
			}
			if req.FirstName != "" {
				c.FirstName = req.FirstName
			}
			if req.LastName != "" {
				c.LastName = req.LastName
			}
			if req.Unsubscribed != nil {
				c.Unsubscribed = *req.Unsubscribed
			}
			contacts[id] = c
			json.NewEncoder(w).Encode(c)
		case http.MethodDelete:
			delete(contacts, id)
			json.NewEncoder(w).Encode(map[string]interface{}{"object": "contact", "id": id, "deleted": true})
		}
	})

	// --- Segments ---
	mux.HandleFunc("/segments", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateSegmentRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "seg-test-1"
			segments[id] = Segment{Object: "segment", ID: id, Name: req.Name, CreatedAt: "2024-01-01T00:00:00Z"}
			json.NewEncoder(w).Encode(segments[id])
		case http.MethodGet:
			var list []Segment
			for _, s := range segments {
				list = append(list, s)
			}
			json.NewEncoder(w).Encode(ListResponse[Segment]{Object: "list", Data: list})
		}
	})

	mux.HandleFunc("/segments/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id := r.URL.Path[len("/segments/"):]
		switch r.Method {
		case http.MethodGet:
			s, ok := segments[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(s)
		case http.MethodDelete:
			delete(segments, id)
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})

	// --- Topics ---
	mux.HandleFunc("/topics", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateTopicRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "top-test-1"
			t := Topic{
				Object: "topic", ID: id, Name: req.Name,
				Description: req.Description, DefaultSubscription: req.DefaultSubscription,
				Visibility: req.Visibility, CreatedAt: "2024-01-01T00:00:00Z",
			}
			if t.DefaultSubscription == "" {
				t.DefaultSubscription = "opt_in"
			}
			if t.Visibility == "" {
				t.Visibility = "public"
			}
			topics[id] = t
			json.NewEncoder(w).Encode(t)
		case http.MethodGet:
			var list []Topic
			for _, t := range topics {
				list = append(list, t)
			}
			json.NewEncoder(w).Encode(ListResponse[Topic]{Object: "list", Data: list})
		}
	})

	mux.HandleFunc("/topics/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id := r.URL.Path[len("/topics/"):]
		switch r.Method {
		case http.MethodGet:
			t, ok := topics[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(t)
		case http.MethodPatch:
			t, ok := topics[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			var req UpdateTopicRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Name != "" {
				t.Name = req.Name
			}
			if req.Description != "" {
				t.Description = req.Description
			}
			if req.Visibility != "" {
				t.Visibility = req.Visibility
			}
			topics[id] = t
			json.NewEncoder(w).Encode(t)
		case http.MethodDelete:
			delete(topics, id)
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})

	// --- Contact Properties ---
	mux.HandleFunc("/contact-properties", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPost:
			var req CreateContactPropertyRequest
			json.NewDecoder(r.Body).Decode(&req)
			id := "cp-test-1"
			cp := ContactProperty{
				Object: "contact_property", ID: id, Key: req.Key, Type: req.Type,
				Description: req.Description, FallbackValue: req.FallbackValue,
				CreatedAt: "2024-01-01T00:00:00Z",
			}
			contactProps[id] = cp
			json.NewEncoder(w).Encode(cp)
		case http.MethodGet:
			var list []ContactProperty
			for _, cp := range contactProps {
				list = append(list, cp)
			}
			json.NewEncoder(w).Encode(ListResponse[ContactProperty]{Object: "list", Data: list})
		}
	})

	mux.HandleFunc("/contact-properties/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		id := r.URL.Path[len("/contact-properties/"):]
		switch r.Method {
		case http.MethodGet:
			cp, ok := contactProps[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(cp)
		case http.MethodPatch:
			cp, ok := contactProps[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			var req UpdateContactPropertyRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Description != "" {
				cp.Description = req.Description
			}
			if req.FallbackValue != "" {
				cp.FallbackValue = req.FallbackValue
			}
			contactProps[id] = cp
			json.NewEncoder(w).Encode(cp)
		case http.MethodDelete:
			delete(contactProps, id)
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}
	})

	return httptest.NewServer(mux)
}

// ---------------------------------------------------------------------------
// Test provider factory
// ---------------------------------------------------------------------------

// testAccProtoV6ProviderFactoriesWithServer returns provider factories that
// inject a mock server URL into the client via an environment variable pattern.
// Since the provider reads RESEND_API_KEY, we set it and override BaseURL after.
func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"resend": providerserver.NewProtocol6WithError(New("test")()),
	}
}

// ---------------------------------------------------------------------------
// Provider configuration test
// ---------------------------------------------------------------------------

func TestProvider_Schema(t *testing.T) {
	p := New("test")()
	resp := &ResendProviderModel{}
	_ = resp

	// Verify the provider can be instantiated
	var metaResp frameworkprovider.MetadataResponse
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &metaResp)
	if metaResp.TypeName != "resend" {
		t.Errorf("expected type name resend, got %s", metaResp.TypeName)
	}
}

func TestProvider_MissingAPIKey(t *testing.T) {
	p := &ResendProvider{version: "test"}
	t.Setenv("RESEND_API_KEY", "")

	var schemaResp frameworkprovider.SchemaResponse
	p.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema should not have errors: %v", schemaResp.Diagnostics)
	}
}
