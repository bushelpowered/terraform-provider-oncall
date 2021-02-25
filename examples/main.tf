terraform {
  required_providers {
    oncall = {
      version = "0.2"
      source  = "github.com/bushelpowered/oncall"
    }
  }
}

provider "oncall" {}
