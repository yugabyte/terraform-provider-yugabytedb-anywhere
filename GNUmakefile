default: fmtTf documents testacc arclint

# Lint test
.PHONY: arclint
arclint:
	arc lint --lintall
# Format terraform files
.PHONY: fmtTf
fmtTf:
	terraform fmt -recursive
# Generate documents
.PHONY: documents
documents:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --rendered-provider-name 'YugabyteDB Anywhere' --provider-name yba
# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: updateclient
updateclient:
	go get github.com/yugabyte/platform-go-client
