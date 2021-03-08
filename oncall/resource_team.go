package oncall

import (
	"context"
	"strings"

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
	teamFieldAdmins             = "admins"
)

func resourceTeam() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceTeamCreate,
		ReadContext:   resourceTeamRead,
		UpdateContext: resourceTeamUpdate,
		DeleteContext: resourceTeamDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceTeamImport,
		},
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
			teamFieldAdmins: &schema.Schema{
				Type:        schema.TypeSet,
				Description: "Authoritative list of usernames of who should admin the team",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceTeamImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	traceLog("Going to import team %s", d.Id())
	var err error

	readErr := resourceTeamRead(ctx, d, m)
	if len(readErr) > 0 {
		err = errors.New(readErr[0].Summary)
	}
	return []*schema.ResourceData{d}, errors.Wrap(err, "Reading team for import")
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
		if strings.Contains(err.Error(), "(422)") {
			return diagFromErrf(err, "Team already exists, please import using id %q", teamConfig.Name)
		}
		return diagFromErrf(err, "Creating oncall team")
	}

	traceLog("Setting team resource id to %q", t.Name)
	d.SetId(t.Name)

	admins := getResourceStringSet(d, teamFieldAdmins)
	err = c.SetTeamAdmins(t.Name, admins)
	if err != nil {
		return diagFromErrf(err, "Setting team admins to %v", admins)
	}

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

	admins := make([]string, 0, len(team.Admins))
	for _, a := range team.Admins {
		admins = append(admins, a.Name)
	}
	setResourceStringSet(d, teamFieldAdmins, admins)

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
