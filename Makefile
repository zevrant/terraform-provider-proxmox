CURRENT_DIR = $(shell pwd)


all: goBuild gotest install e2eTesting reset

gotest:
	bash generateMocks.sh && \
	go test ./...

goBuild:
	go build .

install:
	go install .

e2eTesting: enable-tf-overrides e2e-vms e2e-vm-migrate

enable-tf-overrides:
	cp .terraformrc ~

e2e-vms:
	cd examples/vms && terraform apply -auto-approve
	cd examples/vms && mv main.tf main.bak
	cd examples/vms && mv main-delete-disk.bak main-delete-disk.tf
	cd examples/vms && terraform apply -auto-approve
	cd examples/vms && mv main-delete-disk.tf main-delete-disk.bak
	cd examples/vms && mv main-post-migrate.bak main-post-migrate.tf
	cd examples/vms && terraform apply -auto-approve
	cd examples/vms && terraform destroy -auto-approve
	cd examples/vms && mv main-post-migrate.tf main-post-migrate.bak
	cd examples/vms && mv main.bak main.tf

e2e-vm-migrate:
	cd examples/vmMigration && terraform apply -auto-approve
	cd examples/vmMigration && mv main.tf main.bak
	cd examples/vmMigration && mv main-migrate.bak main-migrate.tf
	cd examples/vmMigration && terraform apply -auto-approve
	cd examples/vmMigration && terraform destroy -auto-approve

reset: e2e-vm-migrate-reset e2e-vms-reset

e2e-vms-reset:
	cd $(CURRENT_DIR)
	cd examples/vms
	cd examples/vms && terraform destroy -auto-approve
	- cd examples/vms && mv main-delete-disk.tf main-delete-disk.bak 2>/dev/null
	- cd examples/vms && mv main-post-migrate.tf main-post-migrate.bak 2>/dev/null
	- cd examples/vms && mv main.bak main.tf


e2e-vm-migrate-reset:
	- cd examples/vmMigration && terraform destroy -auto-approve
	- cd examples/vmMigration && mv main-migrate.tf main-migrate.bak 2>/dev/null
	- cd examples/vmMigration && mv main.bak main.tf