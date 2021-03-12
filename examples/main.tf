terraform {
  required_providers {
    oncall = {
      version = "0.2"
      source  = "github.com/bushelpowered/oncall"
    }
  }
}

provider "oncall" {}

resource "oncall_team" "t" {
  name = "terraform-test-team"

  admins = [
    "oisaac",
  ]
}

resource "oncall_team" "systems" {
  name   = "Systems"
  admins = []
}

resource "oncall_roster" "t" {
  team = oncall_team.t.name
  name = oncall_team.t.name
  members = [
    "oisaac",
    "jbiel",
  ]
}

// 24/7, monday to monday
resource "oncall_basic_schedule" "t" {
  role      = "primary"
  roster_id = oncall_roster.t.id

  auto_populate_days = 21

  start_day_of_week     = "Monday"
  start_time            = "13:00"
  rotate_frequency      = "weekly" // or bi-weekly
  scheduling_algorithim = "default"
}

// Work days, 8-5
resource "oncall_advanced_schedule" "t" {
  role                  = "secondary"
  roster_id             = oncall_roster.t.id
  scheduling_algorithim = "default"
  auto_populate_days    = 21

  shift {
    start_day_of_week = "Monday"
    start_time        = "08:00"
    duration          = "9h"
  }

  shift {
    start_day_of_week = "Tuesday"
    start_time        = "08:00"
    duration          = "9h"
  }

  shift {
    start_day_of_week = "Wednesday"
    start_time        = "08:00"
    duration          = "9h"
  }

  shift {
    start_day_of_week = "Thursday"
    start_time        = "08:00"
    duration          = "9h"
  }

  shift {
    start_day_of_week = "Friday"
    start_time        = "08:00"
    duration          = "9h"
  }
}

