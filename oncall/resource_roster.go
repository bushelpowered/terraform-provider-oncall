package oncall

import (
	"context"
	"fmt"
	"strings"

	"github.com/bushelpowered/oncall-client-go/oncall"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

const (
	rosterFieldName    = "name"
	rosterFieldTeam    = "team"
	rosterFieldMembers = "members"
)

func resourceRoster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRosterCreate,
		ReadContext:   resourceRosterRead,
		UpdateContext: resourceRosterUpdate,
		DeleteContext: resourceRosterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRosterImport,
		},

		Schema: map[string]*schema.Schema{
			rosterFieldName: &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Computed:    true,
				Description: "Name of the roster, if blank will default to team name",
			},
			rosterFieldTeam: &schema.Schema{
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Name of team this roster should be assigned to",
			},
			rosterFieldMembers: &schema.Schema{
				Type:        schema.TypeSet,
				Description: "List of usernames which should be added to the roster",
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceRosterCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	diags := diag.Diagnostics{}
	c := m.(*oncall.Client)

	teamName := d.Get(rosterFieldTeam).(string)
	rosterName := d.Get(rosterFieldName).(string)

	if teamName == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "You must specify a non-empty " + rosterFieldTeam,
		})
	}
	if len(diags) > 0 {
		return diags
	}

	if rosterName == "" {
		rosterName = teamName
	}

	traceLog("Going to create roster: %s/%s", teamName, rosterName)
	roster, err := c.CreateRoster(teamName, rosterName)
	if err != nil {
		if strings.Contains(err.Error(), "(422)") {
			return diagFromErrf(err, "Roster already exists, please import using id '%s'", getRosterID(teamName, rosterName))
		}
		return diagFromErrf(err, "Creating oncall roster")
	}

	traceLog("Setting roster resource id to %q", roster.ID)
	d.SetId(getRosterID(teamName, rosterName))

	traceLog("Getting roster %s/%s requested members", teamName, rosterName)
	members := getResourceStringSet(d, rosterFieldMembers)

	traceLog("Going to set roster %s/%s members to %v", teamName, rosterName, members)
	err = c.SetRosterUsers(teamName, rosterName, members)
	if err != nil {
		return diagFromErrf(err, "Setting roster members")
	}

	resourceRosterRead(ctx, d, m)
	return diags
}

func resourceRosterImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	teamName, rosterName, err := parseRosterID(d.Id())
	if err != nil {
		return nil, errors.Wrap(err, "Parsing roster ID, this is an internal error")
	}

	traceLog("Going to import roster %q as team: %s, roster: %s", d.Id(), teamName, rosterName)
	d.Set(rosterFieldTeam, teamName)
	d.Set(rosterFieldName, rosterName)

	readErr := resourceRosterRead(ctx, d, m)
	if len(readErr) > 0 {
		err = errors.New(readErr[0].Summary)
	}
	return []*schema.ResourceData{d}, errors.Wrap(err, "Reading resource for import")
}

func resourceRosterRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	teamName, rosterName, err := parseRosterID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}

	roster, err := c.GetRoster(teamName, rosterName)
	if err != nil {
		return diagFromErrf(err, "Getting roster %s/%s", teamName, rosterName)
	}

	d.Set(rosterFieldName, roster.Name)

	members := make([]string, 0, len(roster.Users))
	for _, m := range roster.Users {
		members = append(members, m.Name)
	}
	setResourceStringSet(d, rosterFieldMembers, members)

	return diags
}

func resourceRosterUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	traceLog("Going to update roster %q", d.Id())
	teamName, rosterName, err := parseRosterID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}

	traceLog("Getting roster %s/%s requested members", teamName, rosterName)
	members := getResourceStringSet(d, rosterFieldMembers)

	traceLog("Going to set roster %s/%s members to %v", teamName, rosterName, members)
	err = c.SetRosterUsers(teamName, rosterName, members)
	if err != nil {
		return diagFromErrf(err, "Setting roster members")
	}

	return resourceRosterRead(ctx, d, m)
}

func resourceRosterDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	teamName, rosterName, err := parseRosterID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}

	err = c.DeleteRoster(teamName, rosterName)
	if err != nil {
		return diagFromErrf(err, "Deleting roster")
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diag.Diagnostics{}
}

func getRosterID(team, roster string) string {
	return fmt.Sprintf("%s/%s", team, roster)
}

func parseRosterID(rosterID string) (team, roster string, err error) {
	tr := strings.Split(rosterID, "/")
	if len(tr) == 1 {
		errorLog("Giving roster id %q did not match expected team/roster format", rosterID)
		team = tr[0]
		err = errors.New("Only team name found in roster id")
	} else if len(tr) == 2 {
		team = tr[0]
		roster = tr[1]
	} else {
		errorLog("Giving roster id %q did not match expected team/roster format", rosterID)
		err = errors.New("Unparseable roster id")
	}

	if err == nil && (team == "" || roster == "") {
		err = errors.New("Roster ID did not specify both team and roster")
	}
	return
}
