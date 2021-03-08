module github.com/bushelpowered/terraform-provider-oncall

go 1.15

require (
	github.com/bushelpowered/oncall-client-go/oncall v0.0.0
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.4.4
	github.com/pkg/errors v0.9.1
	maze.io/x/duration v0.0.0-20160924141736-faac084b6075
)

replace github.com/bushelpowered/oncall-client-go/oncall v0.0.0 => ../oncall-client-go/oncall
