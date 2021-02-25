package oncall

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bushelpowered/oncall-client-go/oncall"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

var authMethods = []oncall.AuthMethod{
	oncall.AuthMethodAPI,
	oncall.AuthMethodUser,
}

// Provider - returns the oncall provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Oncall endpoint to connect to, everything before '/api/v0'",
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_ENDPOINT", nil),
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username to use when connecting to oncall",
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_USERNAME", nil),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Password to use when connecting to oncall",
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_PASSWORD", nil),
			},
			"auth_type": &schema.Schema{
				Type:        schema.TypeString,
				Default:     string(oncall.AuthMethodUser),
				Description: fmt.Sprintf("Auth method for your username/password; one of: %v", authMethods),
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ONCALL_AUTH_TYPE", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			//	"hashicups_order": resourceOrder(),
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
	endpoint := d.Get("endpoint").(string)
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	requestedAuthMethod := d.Get("auth_method").(oncall.AuthMethod)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	var authMethod oncall.AuthMethod
	for _, m := range authMethods {
		if m == requestedAuthMethod {
			authMethod = m
			break
		}
	}
	if authMethod == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid auth_method specified",
			Detail:   fmt.Sprintf("Auth method of %s is not a valid auth method %v", requestedAuthMethod, authMethods),
		})
		return nil, diags
	}

	oncallClient, err := oncall.New(nil, oncall.Config{
		Endpoint:   endpoint,
		Username:   username,
		Password:   password,
		AuthMethod: authMethod,
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Unable to create oncall client",
			Detail:   errors.Wrap(err, "Creating oncall clietn").Error(),
		})
		return nil, diags
	}

	return oncallClient, diags
}

func leveledLog(level string) func(format string, v ...interface{}) {
	prefix := fmt.Sprintf("[%s] ", strings.ToUpper(level))
	return func(format string, v ...interface{}) {
		log.Printf(prefix+format, v...)
	}
}

var traceLog = leveledLog("trace")
var debugLog = leveledLog("debug")
var infoLog = leveledLog("info")
var warnLog = leveledLog("warn")
var errorLog = leveledLog("error")
