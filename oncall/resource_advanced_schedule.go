package oncall

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bushelpowered/oncall-client-go/oncall"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
	"maze.io/x/duration"
)

const (
	advancedScheduleFieldShift    = "shift"
	advancedScheduleFieldDuration = "duration"
)

func resourceAdvancedSchedule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAdvancedScheduleCreate,
		ReadContext:   resourceAdvancedScheduleRead,
		UpdateContext: resourceAdvancedScheduleUpdate,
		DeleteContext: resourceAdvancedScheduleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceAdvancedScheduleImport,
		},

		Schema: map[string]*schema.Schema{
			scheduleFieldRole: {
				Type:             schema.TypeString,
				ForceNew:         false,
				Required:         true,
				ValidateDiagFunc: validateStringSliceContains(roleNames),
				Description:      fmt.Sprintf("Name of the role, one of %v", roleNames),
			},
			scheduleFieldRosterID: {
				Type:        schema.TypeString,
				ForceNew:    false,
				Required:    true,
				Description: "Roster ID (in team/roster format) to map this schedule to",
			},
			scheduleFieldAutoPopulateDays: {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     21,
				Description: "How many days in advance to plan the schedule",
			},
			scheduleFieldSchedulingAlgorithim: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "default",
				ValidateDiagFunc: validateStringSliceContains(schedulingAlgorithms),
				Description:      fmt.Sprintf("Scheduling algorithim to use, one of: %v", schedulingAlgorithms),
			},
			advancedScheduleFieldShift: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    false,
				Description: "The various shifts that make up a rotation of this role",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						scheduleFieldStartDayOfWeek: {
							Type:             schema.TypeString,
							ValidateDiagFunc: validateStringSliceContains(daysOfWeek),
							Required:         true,
							Description:      "The day of week that this shift should start on",
						},
						scheduleFieldStartTime: {
							Type:             schema.TypeString,
							ValidateDiagFunc: validate24HourTime,
							Required:         true,
							Description:      "The time on this day that this shift should start",
						},
						advancedScheduleFieldDuration: {
							Type:             schema.TypeString,
							ValidateDiagFunc: validateDuration,
							Required:         true,
							Description:      "How long this shift should be in duration shorthand, e.g. 24h, 8h, 1h30m, 3d",
						},
					},
				},
			},
		},
	}
}

func resourceAdvancedScheduleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	diags := diag.Diagnostics{}
	c := m.(*oncall.Client)

	rosterID := d.Get(scheduleFieldRosterID).(string)
	teamName, rosterName, err := parseRosterID(rosterID)
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}
	scheduleName := d.Get(scheduleFieldRole).(string)

	traceLog("Going to create roster schedule: %s/%s/%s", teamName, rosterName, scheduleName)
	sched, err := advancedScheduleFromResource(d)
	if err != nil {
		return diagFromErrf(err, "Failed to parse resource into oncall schedule")
	}

	resourceID := getScheduleID(teamName, rosterName, scheduleName)
	err = c.AddRosterSchedule(teamName, rosterName, sched)
	if err != nil {
		if strings.Contains(err.Error(), "(422)") {
			return diagFromErrf(err, "Roster schedule already exists, please import using id '%s", resourceID)
		}
		return diagFromErrf(err, "Creating oncall roster")
	}

	d.SetId(resourceID)
	resourceAdvancedScheduleRead(ctx, d, m)
	return diags
}

func resourceAdvancedScheduleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	teamName, rosterName, scheduleName, err := parseScheduleID(d.Id())
	if err != nil {
		return nil, errors.Wrap(err, "Parsing roster ID, this is an internal error")
	}

	rosterID := getRosterID(teamName, rosterName)

	traceLog("Going to import roster schedule %q as team: %s, roster: %s, role: ", d.Id(), teamName, rosterName, scheduleName)
	d.Set(scheduleFieldRole, scheduleName)
	d.Set(scheduleFieldRosterID, rosterID)

	readErr := resourceAdvancedScheduleRead(ctx, d, m)
	if len(readErr) > 0 {
		err = errors.New(readErr[0].Summary)
	}
	return []*schema.ResourceData{d}, errors.Wrap(err, "Reading resource for import")
}

func resourceAdvancedScheduleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	teamName, rosterName, scheduleName, err := parseScheduleID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}

	schedule, err := c.GetRosterSchedule(teamName, rosterName, scheduleName)
	if err != nil {
		if strings.Contains(err.Error(), "Did not find schedule") {
			schedule = oncall.Schedule{
				Role: scheduleName,
				Scheduler: oncall.ScheduleScheduler{
					Name: "default",
				},
			}
		} else {
			return diagFromErrf(err, "Getting roster schedule %s/%s/%s", teamName, rosterName, scheduleName)
		}
	}

	d.Set(scheduleFieldRole, schedule.Role)
	d.Set(scheduleFieldRosterID, getRosterID(teamName, rosterName))
	d.Set(scheduleFieldAutoPopulateDays, schedule.AutoPopulateThreshold)
	d.Set(scheduleFieldSchedulingAlgorithim, schedule.Scheduler.Name)

	events := make([]map[string]interface{}, 0, len(schedule.Events))
	for _, event := range schedule.Events {
		dayOfWeekIndex, startHour, startMin := secondsToDayHourMinute(event.Start)
		ev := map[string]interface{}{
			scheduleFieldStartDayOfWeek:   daysOfWeek[dayOfWeekIndex],
			scheduleFieldStartTime:        fmt.Sprintf("%02d:%02d", startHour, startMin),
			advancedScheduleFieldDuration: prettyPrintDuration(event.Duration),
		}
		events = append(events, ev)
	}
	d.Set(advancedScheduleFieldShift, events)
	return diags
}

func resourceAdvancedScheduleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	traceLog("Going to update schedule %q", d.Id())
	teamName, rosterName, schedulename, err := parseScheduleID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster schedule ID, this is an internal error")
	}

	traceLog("Going to update roster schedule %s/%s/%s", teamName, rosterName, schedulename)
	sched, err := advancedScheduleFromResource(d)
	if err != nil {
		return diagFromErrf(err, "Failed to parse resource into oncall schedule")
	}

	err = c.UpdateRosterSchedule(teamName, rosterName, sched.Role, sched)
	if err != nil {
		return diagFromErrf(err, "Updating oncall roster schedule")
	}
	err = c.PopulateRosterSchedule(teamName, rosterName, sched.Role, time.Now())
	if err != nil {
		return diagFromErrf(err, "Populating oncall roster schedule")
	}

	return resourceAdvancedScheduleRead(ctx, d, m)
}

func resourceAdvancedScheduleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	traceLog("Going to update roster %q", d.Id())
	teamName, rosterName, scheduleName, err := parseScheduleID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster schedule ID, this is an internal error")
	}

	traceLog("Going to delete roster schedule %s/%s/%s", teamName, rosterName, scheduleName)
	err = c.RemoveRosterSchedule(teamName, rosterName, scheduleName)
	if err != nil {
		if !strings.Contains(err.Error(), "Did not find schedule") {
			return diagFromErrf(err, "Removing roster %s/%s/%s", teamName, rosterName, scheduleName)
		}
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diag.Diagnostics{}
}

func advancedScheduleFromResource(d *schema.ResourceData) (oncall.Schedule, error) {
	role := d.Get(scheduleFieldRole).(string)
	rosterID := d.Get(scheduleFieldRosterID).(string)
	autoPopulateDays := d.Get(scheduleFieldAutoPopulateDays).(int)
	schedulingAlgorithim := d.Get(scheduleFieldSchedulingAlgorithim).(string)

	sched := oncall.Schedule{
		AdvancedMode:          1,
		Role:                  role,
		AutoPopulateThreshold: autoPopulateDays,
		Scheduler: oncall.ScheduleScheduler{
			Name: schedulingAlgorithim,
		},
	}

	team, roster, err := parseRosterID(rosterID)
	if err != nil {
		return sched, errors.Wrapf(err, "Invalid roster ID %q", rosterID)
	}
	sched.Team = team
	sched.Roster = roster

	shiftInterfaces := d.Get(advancedScheduleFieldShift).([]interface{})

	for _, shiftRaw := range shiftInterfaces {
		shift := shiftRaw.(map[string]interface{})

		durationString := shift[advancedScheduleFieldDuration].(string)
		startDayOfWeek := shift[scheduleFieldStartDayOfWeek].(string)
		startTime := shift[scheduleFieldStartTime].(string)

		startSeconds, err := weekdayStartTimeToSeconds(startDayOfWeek, startTime)
		if err != nil {
			return sched, errors.Wrapf(err, "Parsing start weekday and time")
		}

		duration, err := duration.ParseDuration(durationString)
		if err != nil {
			return sched, errors.Wrapf(err, "Failed to parse duration")
		}
		event := oncall.ScheduleEvent{
			Start:    startSeconds,
			Duration: int(duration.Seconds()),
		}

		sched.Events = append(sched.Events, event)
	}
	return sched, nil
}

func validateDuration(in interface{}, path cty.Path) diag.Diagnostics {
	_, err := duration.ParseDuration(in.(string))
	return diagFromErrf(err, "Failed to parse duration")
}

func prettyPrintDuration(dur int) string {
	numWeeks := int(dur / int(duration.Week.Seconds()))

	durWithoutWeeks := int(dur - numWeeks*int(duration.Week.Seconds()))
	numDays := int(durWithoutWeeks / int(duration.Day.Seconds()))

	durWithoutDays := int(durWithoutWeeks - numDays*int(duration.Day.Seconds()))
	numHours := int(durWithoutDays / int(duration.Hour.Seconds()))

	durWithoutHours := int(durWithoutDays - numHours*int(duration.Hour.Seconds()))
	numMinutes := int(durWithoutHours / int(duration.Minute.Seconds()))

	numSeconds := int(durWithoutHours - numMinutes*int(duration.Minute.Seconds()))

	ret := ""
	if numWeeks > 0 {
		ret = fmt.Sprintf("%d", numWeeks) + "w"
	}

	if numDays > 0 {
		ret = fmt.Sprintf("%s%d", ret, numDays) + "d"
	}

	if numHours > 0 {
		ret = fmt.Sprintf("%s%d", ret, numHours) + "h"
	}

	if numMinutes > 0 {
		ret = fmt.Sprintf("%s%d", ret, numMinutes) + "m"
	}

	if numSeconds > 0 {
		ret = fmt.Sprintf("%s%d", ret, numSeconds) + "s"
	}

	return ret
}
