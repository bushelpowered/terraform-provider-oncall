package oncall

import (
	"context"
	"fmt"

	"github.com/bushelpowered/oncall-client-go/oncall"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

var authMethods = []oncall.AuthMethod{
	oncall.AuthMethodAPI,
	oncall.AuthMethodUser,
}

const (
	providerFieldEndpoint = "endpoint"
	providerFieldUsername = "username"
	providerFieldPassword = "password"
	providerFieldAuthType = "auth_type"
)

// Provider - returns the oncall provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			providerFieldEndpoint: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Oncall endpoint to connect to, everything before '/api/v0' in the URL",
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_ENDPOINT", ""),
			},
			providerFieldUsername: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username to use when connecting to oncall",
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_USERNAME", ""),
			},
			providerFieldPassword: {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password to use when connecting to oncall",
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_PASSWORD", ""),
			},
			providerFieldAuthType: {
				Type:        schema.TypeString,
				Default:     string(oncall.AuthMethodUser),
				Description: fmt.Sprintf("Auth method for your username/password; one of: %v", authMethods),
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_AUTH_TYPE", ""),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"oncall_team":              resourceTeam(),
			"oncall_roster":            resourceRoster(),
			"oncall_basic_schedule":    resourceBasicSchedule(),
			"oncall_advanced_schedule": resourceAdvancedSchedule(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			//	"hashicups_coffees":     dataSourceCoffees(),
			//	"hashicups_ingredients": dataSourceIngredients(),
			//	"hashicups_order":       dataSourceOrder(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	endpoint := d.Get(providerFieldEndpoint).(string)
	username := d.Get(providerFieldUsername).(string)
	password := d.Get(providerFieldPassword).(string)
	requestedAuthMethod := d.Get(providerFieldAuthType).(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	var authMethod oncall.AuthMethod
	for _, m := range authMethods {
		if m == oncall.AuthMethod(requestedAuthMethod) {
			authMethod = m
			break
		}
	}
	if authMethod == "" {
		return nil, diag.FromErr(fmt.Errorf("%s of %s is not valid, must be one of: %v", providerFieldAuthType, requestedAuthMethod, authMethods))
	}

	traceLog("Going to create oncall client for %s with auth method %s, username %s", endpoint, authMethod, username)

	oncallClient, err := oncall.New(nil, oncall.Config{
		Endpoint:   endpoint,
		Username:   username,
		Password:   password,
		AuthMethod: authMethod,
	})
	if err != nil {
		return nil, diag.FromErr(errors.Wrap(err, "Initializing oncall client"))
	}

	return oncallClient, diags
}
