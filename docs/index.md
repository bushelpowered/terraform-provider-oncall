---
page_title: "Provider: Oncall"
subcategory: ""
description: |-
  Terraform provider for interacting with the Oncall API.
---

# Oncall Provider

The Oncall provider is used to interact with the Oncall service you are running. Details about runnign oncall can be found at [oncall.tools](https://oncall.tools).

## Example Usage

There are two ways to authenticate with oncall, as a user or as an API client. 

**API Clients** are not able to create Teams, however User's are.

Do not keep your authentication password in HCL for production environments, use Terraform environment variables.

```terraform
// API Client
provider "oncall" {
    auth_type = "api"
    endpoint = "https://example.com/oncall/"
    username = "terraform_user"
    password = "password123"
}

// User Client
provider "oncall" {
    auth_type = "user"
    endpoint = "https://example.com/oncall/"
    username = "firstname.lastname" // If you are using ldap to login, this is the ldap username
    password = "password123" 
}
```

## Schema

### Optional

- **endpoint** (String, Optional) Oncall endpoint to connect to, everything before `/api/v0` in the URL
- **auth_type** (String, Optional) Auth method for your username/password; one of: `api` or `user`
- **username** (String, Optional) Username to use when connecting to oncall
- **password** (String, Optional) Password to use when connecting to oncall
