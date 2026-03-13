terraform {
  required_providers {
    resend = {
      source = "socle-labs/resend"
    }
  }
}

provider "resend" {
  # Set via RESEND_API_KEY environment variable, or:
  # api_key = "re_xxxxxxxxx"
}

# --------------------------------------------------------------------------
# Domain
# --------------------------------------------------------------------------

resource "resend_domain" "primary" {
  name   = "example.com"
  region = "us-east-1"

  open_tracking  = true
  click_tracking = false
  tls            = "enforced"
}

# --------------------------------------------------------------------------
# API Key
# --------------------------------------------------------------------------

resource "resend_api_key" "sending" {
  name       = "production-sending"
  permission = "sending_access"
  domain_id  = resend_domain.primary.id
}

resource "resend_api_key" "admin" {
  name       = "admin-full-access"
  permission = "full_access"
}

# --------------------------------------------------------------------------
# Segment (formerly Audience)
# --------------------------------------------------------------------------

resource "resend_segment" "newsletter" {
  name = "Newsletter Subscribers"
}

resource "resend_segment" "beta_users" {
  name = "Beta Users"
}

# --------------------------------------------------------------------------
# Topic
# --------------------------------------------------------------------------

resource "resend_topic" "weekly" {
  name                 = "Weekly Newsletter"
  description          = "Weekly product updates and tips"
  default_subscription = "opt_in"
  visibility           = "public"
}

resource "resend_topic" "changelog" {
  name                 = "Changelog"
  description          = "Product changelog and release notes"
  default_subscription = "opt_out"
  visibility           = "public"
}

# --------------------------------------------------------------------------
# Contact Property
# --------------------------------------------------------------------------

resource "resend_contact_property" "company" {
  key           = "company_name"
  type          = "string"
  description   = "The company the contact works at"
  fallback_value = "Unknown"
}

resource "resend_contact_property" "plan" {
  key           = "plan_tier"
  type          = "string"
  description   = "Subscription plan tier"
  fallback_value = "free"
}

# --------------------------------------------------------------------------
# Contacts
# --------------------------------------------------------------------------

resource "resend_contact" "alice" {
  email      = "alice@example.com"
  first_name = "Alice"
  last_name  = "Smith"

  properties = {
    company_name = "Acme Corp"
    plan_tier    = "pro"
  }

  depends_on = [
    resend_contact_property.company,
    resend_contact_property.plan,
  ]
}

resource "resend_contact" "bob" {
  email      = "bob@example.com"
  first_name = "Bob"
  last_name  = "Jones"
}

# --------------------------------------------------------------------------
# Webhook
# --------------------------------------------------------------------------

resource "resend_webhook" "events" {
  endpoint = "https://example.com/webhooks/resend"
  events   = [
    "email.sent",
    "email.delivered",
    "email.bounced",
    "email.complained",
    "email.opened",
    "email.clicked",
  ]
}

# --------------------------------------------------------------------------
# Data Sources — Singular (look up by ID)
# --------------------------------------------------------------------------

data "resend_domain" "existing" {
  id = "d91cd9bd-1176-453e-8fc1-35364d380206"
}

data "resend_webhook" "existing" {
  id = "4dd369bc-aa82-4ff3-97de-514ae3000ee0"
}

data "resend_contact" "existing" {
  id = "e169aa45-1ecf-4183-9955-b1499d5701d3"
}

data "resend_segment" "existing" {
  id = "78261eea-8f8b-4381-83c6-79fa7120f1cf"
}

data "resend_topic" "existing" {
  id = "b6d24b8e-af0b-4c3c-be0c-359bbd97381e"
}

data "resend_contact_property" "existing" {
  id = "b6d24b8e-af0b-4c3c-be0c-359bbd97381e"
}

# --------------------------------------------------------------------------
# Data Sources — List all
# --------------------------------------------------------------------------

data "resend_domains" "all" {}
data "resend_api_keys" "all" {}
data "resend_webhooks" "all" {}
data "resend_contacts" "all" {}
data "resend_segments" "all" {}
data "resend_topics" "all" {}
data "resend_contact_properties" "all" {}

# --------------------------------------------------------------------------
# Outputs
# --------------------------------------------------------------------------

output "domain_id" {
  value = resend_domain.primary.id
}

output "domain_status" {
  value = resend_domain.primary.status
}

output "sending_api_key_token" {
  value     = resend_api_key.sending.token
  sensitive = true
}

output "webhook_signing_secret" {
  value     = resend_webhook.events.signing_secret
  sensitive = true
}

output "all_domain_names" {
  value = data.resend_domains.all.domains[*].name
}

output "all_topic_names" {
  value = data.resend_topics.all.topics[*].name
}

output "existing_domain_status" {
  value = data.resend_domain.existing.status
}
