package oncall

import (
	"context"

	"github.com/bushelpowered/oncall-client-go/oncall"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

const (
	teamFieldName               = "name"
	teamFieldSchedulingTimezone = "scheduling_timezone"
	teamFieldEmail              = "email"
	teamFieldSlackChannel       = "slack_channel"
	teamFieldIrisPlan           = "iris_plan"
)

func resourceTeam() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTeamCreate,
		ReadContext:   resourceTeamRead,
		UpdateContext: resourceTeamUpdate,
		DeleteContext: resourceTeamDelete,
		Schema: map[string]*schema.Schema{
			teamFieldName: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the team, acts as the ID as well",
			},
			teamFieldSchedulingTimezone: &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "US/Central",
				Description: "Must be non-empty. Scheduling timezone of the team, should be one of values set in your oncall config -> supported_timezones : https://github.com/linkedin/oncall/blob/master/configs/config.yaml#L128-L137",
			},
			teamFieldEmail: &schema.Schema{
				Type:        schema.TypeString,
				Description: "Email group for the entire team",
				Optional:    true,
			},
			teamFieldSlackChannel: &schema.Schema{
				Type:        schema.TypeString,
				Description: "Slack channel that this team should all be members of",
				Optional:    true,
			},
			teamFieldIrisPlan: &schema.Schema{
				Type:        schema.TypeString,
				Description: "Default iris plan for this team. Allows paging from oncall",
				Optional:    true,
			},
		},
	}
}

func resourceTeamCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	// Warning or errors can be collected in a slice type
	teamConfig, diags := resourceTeamAsTeamConfig(d)
	if len(diags) > 0 {
		return diags
	}

	traceLog("Going to create team: %+v", teamConfig)
	t, err := c.CreateTeam(teamConfig)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "Creating oncall team"))
	}

	traceLog("Setting team resource id to %q", t.Name)
	d.SetId(t.Name)

	resourceTeamRead(ctx, d, m)
	return diags
}

func resourceTeamAsTeamConfig(d *schema.ResourceData) (oncall.TeamConfig, diag.Diagnostics) {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	teamConfig := oncall.TeamConfig{
		Name:               d.Get(teamFieldName).(string),
		SchedulingTimezone: d.Get(teamFieldSchedulingTimezone).(string),
		Email:              d.Get(teamFieldEmail).(string),
		SlackChannel:       d.Get(teamFieldSlackChannel).(string),
		IrisPlan:           d.Get(teamFieldIrisPlan).(string),
	}

	if teamConfig.Name == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "You must specify a non-empty " + teamFieldName,
		})
	}

	if teamConfig.SchedulingTimezone == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "You must specify a non-empty " + teamFieldSchedulingTimezone,
		})
	}

	return teamConfig, diags
}

func resourceTeamRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	teamName := d.Id()
	team, err := c.GetTeam(teamName)
	if err != nil {
		return diag.FromErr(errors.Wrapf(err, "Fetching team %s", teamName))
	}

	d.Set(teamFieldName, team.Name)
	d.Set(teamFieldEmail, team.Email)
	d.Set(teamFieldSlackChannel, team.SlackChannel)
	d.Set(teamFieldIrisPlan, team.IrisPlan)
	d.Set(teamFieldSchedulingTimezone, team.SchedulingTimezone)

	return diags
}

func resourceTeamUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	// Warning or errors can be collected in a slice type
	teamConfig, diags := resourceTeamAsTeamConfig(d)
	if len(diags) > 0 {
		return diags
	}

	traceLog("Going to update team %q: %+v", d.Id(), teamConfig)
	t, err := c.UpdateTeam(d.Id(), teamConfig)
	if err != nil {
		return diag.FromErr(errors.Wrap(err, "Updating oncall team"))
	}

	traceLog("Setting team resource id to %q", t.Name)
	d.SetId(t.Name)

	return resourceTeamRead(ctx, d, m)
}

func resourceTeamDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)
	err := c.DeleteTeam(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diag.Diagnostics{}
}
