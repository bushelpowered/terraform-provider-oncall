package oncall

import (
	"context"
	"fmt"
	"math"
	"strconv"
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
	// Used by basic and advanced schedule
	scheduleFieldRole                 = "role"
	scheduleFieldRosterID             = "roster_id"
	scheduleFieldAutoPopulateDays     = "auto_populate_days"
	scheduleFieldStartDayOfWeek       = "start_day_of_week"
	scheduleFieldStartTime            = "start_time"
	scheduleFieldSchedulingAlgorithim = "scheduling_algorithim"

	basicScheduleRotationWeekly   = "weekly"
	basicScheduleRotationBiWeekly = "bi-weekly"

	schedulingAlgorithmDefault    = "default"
	schedulingAlgorithmRoundRobin = "round-robin"

	// Used only by basic schedule
	basicScheduleFieldRotateFrequency = "rotate_frequency"
)

var basicScheduleRotations = []string{
	basicScheduleRotationWeekly,
	basicScheduleRotationBiWeekly,
}

var schedulingAlgorithms = []string{
	schedulingAlgorithmDefault,
	schedulingAlgorithmRoundRobin,
}

var roleNames = []string{
	"primary",
	"secondary",
	"shadow",
	"manager",
	"vacation",
	"unavailable",
}

var daysOfWeek = []string{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

func resourceBasicSchedule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBasicScheduleCreate,
		ReadContext:   resourceBasicScheduleRead,
		UpdateContext: resourceBasicScheduleUpdate,
		DeleteContext: resourceBasicScheduleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceBasicScheduleImport,
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
			scheduleFieldStartDayOfWeek: {
				Type:             schema.TypeString,
				ForceNew:         false,
				Required:         true,
				ValidateDiagFunc: validateStringSliceContains(daysOfWeek),
				Description:      fmt.Sprintf("Day of week to start the schedule one, one of: %v", daysOfWeek),
			},
			scheduleFieldStartTime: {
				Type:             schema.TypeString,
				ForceNew:         false,
				ValidateDiagFunc: validate24HourTime,
				Required:         true,
				Description:      "Start time of schedule in 24 hour time format, e.g. 13:15 for 1:15pm",
			},
			basicScheduleFieldRotateFrequency: {
				Type:             schema.TypeString,
				ForceNew:         false,
				Optional:         true,
				Default:          basicScheduleRotationWeekly,
				ValidateDiagFunc: validateStringSliceContains(basicScheduleRotations),
				Description:      fmt.Sprintf("Rotation frequency, one of: %v", basicScheduleRotations),
			},
			scheduleFieldSchedulingAlgorithim: {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "default",
				ValidateDiagFunc: validateStringSliceContains(schedulingAlgorithms),
				Description:      fmt.Sprintf("Scheduling algorithim to use, one of: %v", schedulingAlgorithms),
			},
		},
	}
}

func resourceBasicScheduleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	diags := diag.Diagnostics{}
	c := m.(*oncall.Client)

	rosterID := d.Get(scheduleFieldRosterID).(string)
	teamName, rosterName, err := parseRosterID(rosterID)
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}
	scheduleName := d.Get(scheduleFieldRole).(string)

	traceLog("Going to create roster schedule: %s/%s/%s", teamName, rosterName, scheduleName)
	sched, err := basicScheduleFromResource(d)
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
	resourceBasicScheduleRead(ctx, d, m)
	return diags
}

func resourceBasicScheduleImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	teamName, rosterName, scheduleName, err := parseScheduleID(d.Id())
	if err != nil {
		return nil, errors.Wrap(err, "Parsing roster ID, this is an internal error")
	}

	rosterID := getRosterID(teamName, rosterName)

	traceLog("Going to import roster schedule %q as team: %s, roster: %s, role: ", d.Id(), teamName, rosterName, scheduleName)
	d.Set(scheduleFieldRole, scheduleName)
	d.Set(scheduleFieldRosterID, rosterID)

	readErr := resourceBasicScheduleRead(ctx, d, m)
	if len(readErr) > 0 {
		err = errors.New(readErr[0].Summary)
	}
	return []*schema.ResourceData{d}, errors.Wrap(err, "Reading resource for import")
}

func resourceBasicScheduleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	teamName, rosterName, scheduleName, err := parseScheduleID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster ID, this is an internal error")
	}

	schedule, err := c.GetRosterSchedule(teamName, rosterName, scheduleName)
	if err != nil {
		return diagFromErrf(err, "Getting roster schedule %s/%s/%s", teamName, rosterName, scheduleName)
	}

	d.Set(scheduleFieldRole, schedule.Role)
	d.Set(scheduleFieldRosterID, getRosterID(teamName, rosterName))
	d.Set(scheduleFieldAutoPopulateDays, schedule.AutoPopulateThreshold)
	d.Set(scheduleFieldSchedulingAlgorithim, schedule.Scheduler.Name)

	if len(schedule.Events) != 1 {
		return diag.Errorf("The schedule you are reading is not a basic schedule as it does not have exactly one event")
	}

	d.Set(basicScheduleFieldRotateFrequency, basicScheduleRotationWeekly)
	if schedule.Events[0].Duration == int(duration.Fortnight.Seconds()) {
		d.Set(basicScheduleFieldRotateFrequency, basicScheduleRotationBiWeekly)
	}

	dayOfWeekIndex, startHour, startMin := secondsToDayHourMinute(schedule.Events[0].Start)
	d.Set(scheduleFieldStartDayOfWeek, daysOfWeek[dayOfWeekIndex])
	d.Set(scheduleFieldStartTime, fmt.Sprintf("%02d:%02d", startHour, startMin))

	return diags
}

func resourceBasicScheduleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	traceLog("Going to update schedule %q", d.Id())
	teamName, rosterName, schedulename, err := parseScheduleID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster schedule ID, this is an internal error")
	}

	traceLog("Going to update roster schedule %s/%s/%s", teamName, rosterName, schedulename)
	sched, err := basicScheduleFromResource(d)
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

	return resourceBasicScheduleRead(ctx, d, m)
}

func resourceBasicScheduleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*oncall.Client)

	traceLog("Going to update roster %q", d.Id())
	teamName, rosterName, scheduleName, err := parseScheduleID(d.Id())
	if err != nil {
		return diagFromErrf(err, "Parsing roster schedule ID, this is an internal error")
	}

	traceLog("Going to delete roster schedule %s/%s/%s", teamName, rosterName, scheduleName)
	err = c.RemoveRosterSchedule(teamName, rosterName, scheduleName)
	if err != nil {
		return diagFromErrf(err, "Removing roster %s/%s/%s", teamName, rosterName, scheduleName)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diag.Diagnostics{}
}

func getScheduleID(team, roster, role string) string {
	return fmt.Sprintf("%s/%s/%s", team, roster, role)
}

func parseScheduleID(basicScheduleID string) (team, roster, role string, err error) {
	tr := strings.Split(basicScheduleID, "/")
	if len(tr) == 3 {
		team, roster, role = tr[0], tr[1], tr[2]
	} else {
		err = errors.New("Unparseable roster schedule id (should be team/roster/role)")
	}

	if err == nil && (team == "" || roster == "" || role == "") {
		err = errors.New("Roster ID did not specify team, roster, and role")
	}
	return
}

func validate24HourTime(in interface{}, path cty.Path) diag.Diagnostics {
	_, _, err := parseHourMinStr(in.(string))
	if err != nil {
		return diagFromErrf(err, "Invalid HH:MM entry")
	}

	return nil
}

func parseHourMinStr(hourMin string) (hours, minutes int, err error) {
	splitTime := strings.Split(hourMin, ":")
	if len(splitTime) != 2 {
		err = fmt.Errorf("Provided time must be in 24 hour format: HH:MM")
		return
	}

	hourString := strings.TrimLeft(splitTime[0], "0")
	if hourString == "" {
		hourString = "0"
	}

	minString := strings.TrimLeft(splitTime[1], "0")
	if minString == "" {
		minString = "0"
	}

	hours, err = strconv.Atoi(hourString)
	if err != nil {
		err = errors.Wrap(err, "The part of your time before the colon is not a number")
		return
	}

	minutes, err = strconv.Atoi(minString)
	if err != nil {
		err = errors.Wrap(err, "The part of your time after the colon is not a number")
		return
	}

	if hours < 0 || hours >= 24 {
		err = fmt.Errorf("Your provided hours must be 0 - 23")
		return
	}

	if minutes < 0 || minutes >= 60 {
		err = fmt.Errorf("Your provided minutes must be 0 - 59")
		return
	}

	return
}

func basicScheduleFromResource(d *schema.ResourceData) (oncall.Schedule, error) {
	role := d.Get(scheduleFieldRole).(string)
	rosterID := d.Get(scheduleFieldRosterID).(string)
	autoPopulateDays := d.Get(scheduleFieldAutoPopulateDays).(int)
	startDayOfWeek := d.Get(scheduleFieldStartDayOfWeek).(string)
	startTime := d.Get(scheduleFieldStartTime).(string)
	rotateFrequency := d.Get(basicScheduleFieldRotateFrequency).(string)
	schedulingAlgorithim := d.Get(scheduleFieldSchedulingAlgorithim).(string)

	sched := oncall.Schedule{
		AdvancedMode:          0,
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

	dur := duration.Week
	if rotateFrequency == basicScheduleRotationBiWeekly {
		dur = duration.Fortnight
	}

	startSeconds, err := weekdayStartTimeToSeconds(startDayOfWeek, startTime)
	if err != nil {
		return sched, errors.Wrapf(err, "Parsing start weekday and time")
	}
	event := oncall.ScheduleEvent{
		Start:    startSeconds,
		Duration: int(dur.Seconds()),
	}

	sched.Events = append(sched.Events, event)

	return sched, nil
}

func secondsToDayHourMinute(seconds int) (days, hours, minutes int) {
	days = int(math.Floor(float64(seconds / int(duration.Day.Seconds()))))

	timeInDay := seconds % int(duration.Day.Seconds())
	hours = int(math.Floor(float64(timeInDay / int(duration.Hour.Seconds()))))
	minutes = int(math.Floor(float64(timeInDay % int(duration.Hour.Seconds()) / int(duration.Minute.Seconds()))))
	return
}

func weekdayStartTimeToSeconds(weekday, startTime string) (seconds int, err error) {
	hour, min, err := parseHourMinStr(startTime)
	if err != nil {
		return -1, errors.Wrapf(err, "Failed to parse HH:MM input of %q", startTime)
	}

	numDays := -1
	for dayIndex, day := range daysOfWeek {
		if strings.ToLower(day) == strings.ToLower(weekday) {
			numDays = dayIndex
			break
		}
	}
	if numDays == -1 {
		return -1, fmt.Errorf("You did not specify a valid day name")
	}

	return (numDays*int(duration.Day.Seconds()) +
		hour*int(duration.Hour.Seconds()) +
		min*int(duration.Minute.Seconds())), nil
}
