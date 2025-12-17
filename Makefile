CURRENT_DIR = $(shell pwd)


all: goBuild gotest install e2eTesting

gotest:
	bash generateMocks.sh && \
	go test ./...

goBuild:
	go build .

install:
	go install .

e2eTesting: enable-tf-overrides e2e-vms

enable-tf-overrides:
	cp .terraformrc ~

e2e-vms:
	cd $(CURRENT_DIR)
	cd examples/vms
	terraform apply -auto-approve
	mv main.tf main.bak
	mv main-delete-disk.bak main-delete-disk.tf
	terraform apply -auto-approve
	mv main-delete-disk.tf main-delete-disk.bak
	mv main-post-migrate.bak main-post-migrate.tf
	terraform apply -auto-approve
	mv main-post-migrate.tf main-post-migrate.bak
	mv main.bak main.tf
	terraform destroy -auto-approve


e2e-vms-reset:
	cd $(CURRENT_DIR)
	cd examples/vms
	terraform destroy -auto-approve
	- mv main-delete-disk.tf main-delete-disk.bak 2>/dev/null
	- mv main-post-migrate.tf main-post-migrate.bak 2>/dev/null
	- mv main.bak main.tf 2>/dev/null
