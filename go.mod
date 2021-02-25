module github.com/bushelpowered/terraform-provider-oncall

go 1.15

require (
	github.com/bushelpowered/oncall-client-go/oncall v0.0.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.4.4
	github.com/pkg/errors v0.9.1
)

replace github.com/bushelpowered/oncall-client-go/oncall v0.0.0 => ../oncall-client-go/oncall
