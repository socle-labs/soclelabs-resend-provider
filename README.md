# Terraform Provider for Resend

A Terraform provider for managing [Resend](https://resend.com) email infrastructure as code — domains, API keys, webhooks, contacts, segments, topics, and contact properties.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (to build from source)

## Installation

### From source

```bash
git clone https://github.com/socle-labs/terraform-provider-resend.git
cd terraform-provider-resend
go build -o terraform-provider-resend .
```

Move the binary to your Terraform plugin directory:

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/socle-labs/resend/0.1.0/$(go env GOOS)_$(go env GOARCH)
mv terraform-provider-resend ~/.terraform.d/plugins/registry.terraform.io/socle-labs/resend/0.1.0/$(go env GOOS)_$(go env GOARCH)/
```

### Local development override

Add the following to your `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "socle-labs/resend" = "/path/to/your/binary/directory"
  }
  direct {}
}
```

## Authentication

The provider requires a Resend API key. You can provide it in two ways:

1. **Environment variable** (recommended):

```bash
export RESEND_API_KEY="re_xxxxxxxxx"
```

2. **Provider configuration**:

```hcl
provider "resend" {
  api_key = "re_xxxxxxxxx"
}
```

## Usage

```hcl
terraform {
  required_providers {
    resend = {
      source = "socle-labs/resend"
    }
  }
}

provider "resend" {}

# Create a sending domain
resource "resend_domain" "primary" {
  name   = "example.com"
  region = "us-east-1"

  open_tracking  = true
  click_tracking = false
  tls            = "enforced"
}

# Create a scoped API key
resource "resend_api_key" "sending" {
  name       = "production-sending"
  permission = "sending_access"
  domain_id  = resend_domain.primary.id
}

# Set up a webhook
resource "resend_webhook" "events" {
  endpoint = "https://example.com/webhooks/resend"
  events   = ["email.sent", "email.delivered", "email.bounced"]
}

# Create a segment and contacts
resource "resend_segment" "newsletter" {
  name = "Newsletter Subscribers"
}

resource "resend_contact" "alice" {
  email      = "alice@example.com"
  first_name = "Alice"
  last_name  = "Smith"
}

# Create a subscription topic
resource "resend_topic" "weekly" {
  name                 = "Weekly Newsletter"
  description          = "Weekly product updates and tips"
  default_subscription = "opt_in"
  visibility           = "public"
}

# Define a custom contact property
resource "resend_contact_property" "company" {
  key            = "company_name"
  type           = "string"
  description    = "The company the contact works at"
  fallback_value = "Unknown"
}

# List all domains
data "resend_domains" "all" {}
```

## Resources

| Resource | Description |
|---|---|
| `resend_domain` | Manage sending domains with tracking and TLS configuration |
| `resend_api_key` | Create and manage API keys with scoped permissions |
| `resend_webhook` | Configure webhooks for email event notifications |
| `resend_contact` | Manage contacts with custom properties |
| `resend_segment` | Organize contacts into segments (formerly audiences) |
| `resend_topic` | Define subscription topics for email preferences |
| `resend_contact_property` | Create custom contact attributes |

## Data Sources

### Singular (look up by ID)

| Data Source | Description |
|---|---|
| `resend_domain` | Look up a domain by ID |
| `resend_webhook` | Look up a webhook by ID |
| `resend_contact` | Look up a contact by ID |
| `resend_segment` | Look up a segment by ID |
| `resend_topic` | Look up a topic by ID |
| `resend_contact_property` | Look up a contact property by ID |

### List (fetch all)

| Data Source | Description |
|---|---|
| `resend_domains` | List all domains |
| `resend_api_keys` | List all API keys |
| `resend_webhooks` | List all webhooks |
| `resend_contacts` | List all contacts |
| `resend_segments` | List all segments |
| `resend_topics` | List all topics |
| `resend_contact_properties` | List all contact properties |

## Resource Reference

### resend_domain

```hcl
resource "resend_domain" "example" {
  name               = "example.com"         # Required, forces replacement
  region             = "us-east-1"           # Optional: us-east-1, eu-west-1, sa-east-1, ap-northeast-1
  custom_return_path = "send"                # Optional, forces replacement
  open_tracking      = true                  # Optional
  click_tracking     = false                 # Optional
  tls                = "enforced"            # Optional: enforced, opportunistic
}
```

**Read-only attributes:** `id`, `status`, `created_at`

**Import:** `terraform import resend_domain.example <domain_id>`

### resend_api_key

```hcl
resource "resend_api_key" "example" {
  name       = "my-api-key"                  # Required, forces replacement
  permission = "sending_access"              # Optional: full_access, sending_access
  domain_id  = resend_domain.example.id      # Optional, only for sending_access
}
```

**Read-only attributes:** `id`, `token` (sensitive), `created_at`

> **Note:** API keys are immutable — any change forces replacement. The `token` is only available after creation and stored in state.

### resend_webhook

```hcl
resource "resend_webhook" "example" {
  endpoint = "https://example.com/handler"   # Required
  events   = ["email.sent", "email.bounced"] # Required
}
```

**Available events:** `email.sent`, `email.delivered`, `email.delivery_delayed`, `email.bounced`, `email.complained`, `email.opened`, `email.clicked`, `email.received`, `contact.created`, `contact.updated`, `contact.deleted`, `domain.created`, `domain.updated`, `domain.deleted`

**Read-only attributes:** `id`, `signing_secret` (sensitive), `created_at`

**Import:** `terraform import resend_webhook.example <webhook_id>`

### resend_contact

```hcl
resource "resend_contact" "example" {
  email        = "user@example.com"          # Required
  first_name   = "Jane"                      # Optional
  last_name    = "Doe"                       # Optional
  unsubscribed = false                       # Optional, default: false
  properties   = {                           # Optional
    company_name = "Acme Corp"
  }
}
```

**Read-only attributes:** `id`, `created_at`

**Import:** `terraform import resend_contact.example <contact_id>`

### resend_segment

```hcl
resource "resend_segment" "example" {
  name = "VIP Customers"                     # Required, forces replacement
}
```

**Read-only attributes:** `id`, `created_at`

**Import:** `terraform import resend_segment.example <segment_id>`

### resend_topic

```hcl
resource "resend_topic" "example" {
  name                 = "Product Updates"   # Required
  description          = "Monthly updates"   # Optional
  default_subscription = "opt_in"            # Optional: opt_in, opt_out (forces replacement)
  visibility           = "public"            # Optional: public, private
}
```

**Read-only attributes:** `id`, `created_at`

**Import:** `terraform import resend_topic.example <topic_id>`

### resend_contact_property

```hcl
resource "resend_contact_property" "example" {
  key            = "plan_tier"               # Required, forces replacement
  type           = "string"                  # Required: string, number, boolean (forces replacement)
  description    = "Subscription tier"       # Optional
  fallback_value = "free"                    # Optional
}
```

**Read-only attributes:** `id`, `created_at`

**Import:** `terraform import resend_contact_property.example <property_id>`

## Development

```bash
# Build
go build -o terraform-provider-resend .

# Run tests
go test ./...

# Install locally for testing
go install .
```

## License

[MIT](./LICENSE)
