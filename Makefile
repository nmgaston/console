include .env
export

LOCAL_BIN:=$(CURDIR)/bin
PATH:=$(LOCAL_BIN):$(PATH)

# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

compose-up: ### Run docker compose
	docker compose up --build -d postgres && docker compose logs -f
.PHONY: compose-up

compose-up-integration-test: ### Run docker compose with integration test
	docker compose up --build --abort-on-container-exit --exit-code-from integration
.PHONY: compose-up-integration-test

compose-down: ### Down docker compose
	docker compose down --remove-orphans
.PHONY: compose-down

run: ### run app
	go mod tidy && go mod download && \
	GIN_MODE=debug CGO_ENABLED=0 go run ./cmd/app
.PHONY: run

docker-rm-volume: ### remove docker volume
	docker volume rm go-clean-template_pg-data
.PHONY: docker-rm-volume

linter-golangci: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

linter-hadolint: ### check by hadolint linter
	git ls-files --exclude='Dockerfile*' --ignored | xargs hadolint
.PHONY: linter-hadolint

linter-dotenv: ### check by dotenv linter
	dotenv-linter
.PHONY: linter-dotenv

test: ### run test
	go test -v -cover -race ./...
.PHONY: test

integration-test: ### run integration-test
	go clean -testcache && go test -v ./integration-test/...
.PHONY: integration-test

mock: ### run mockgen
	mockgen -source ./internal/usecase/ciraconfigs/interfaces.go        -package mocks  -mock_names Repository=MockCIRAConfigsRepository,Feature=MockCIRAConfigsFeature > ./internal/mocks/ciraconfigs_mocks.go
	mockgen -source ./internal/usecase/devices/interfaces.go            -package mocks  -mock_names Repository=MockDeviceManagementRepository,Feature=MockDeviceManagementFeature > ./internal/mocks/devicemanagement_mocks.go
	mockgen -source ./internal/usecase/amtexplorer/interfaces.go        -package mocks  -mock_names Repository=MockAMTExplorerRepository,Feature=MockAMTExplorerFeature,WSMAN=MockAMTExplorerWSMAN > ./internal/mocks/amtexplorer_mocks.go
	mockgen -source ./internal/usecase/devices/wsman/interfaces.go      -package mocks  > ./internal/mocks/wsman_mocks.go
	mockgen -source ./internal/usecase/export/interface.go              -package mocks  > ./internal/mocks/export_mocks.go
	mockgen -source ./internal/usecase/domains/interfaces.go            -package mocks  -mock_names Repository=MockDomainsRepository,Feature=MockDomainsFeature > ./internal/mocks/domains_mocks.go
	mockgen -source ./internal/controller/ws/v1/interface.go            -package mocks  > ./internal/mocks/wsv1_mocks.go
	mockgen -source ./pkg/logger/logger.go                              -package mocks  -mock_names Interface=MockLogger  > ./internal/mocks/logger_mocks.go
	mockgen -source ./internal/usecase/ieee8021xconfigs/interfaces.go   -package mocks  -mock_names Repository=MockIEEE8021xConfigsRepository,Feature=MockIEEE8021xConfigsFeature > ./internal/mocks/ieee8021xconfigs_mocks.go
	mockgen -source ./internal/usecase/profiles/interfaces.go           -package mocks  -mock_names Repository=MockProfilesRepository,Feature=MockProfilesFeature > ./internal/mocks/profiles_mocks.go
	mockgen -source ./internal/usecase/wificonfigs/interfaces.go        -package mocks  -mock_names Repository=MockWiFiConfigsRepository,Feature=MockWiFiConfigsFeature > ./internal/mocks/wificonfigs_mocks.go
	mockgen -source ./internal/usecase/profilewificonfigs/interfaces.go -package mocks  -mock_names Repository=MockProfileWiFiConfigsRepository,Feature=MockProfileWiFiConfigsFeature > ./internal/mocks/profileswificonfigs_mocks.go
	mockgen -source ./internal/app/interface.go                         -package mocks  > ./internal/mocks/app_mocks.go


.PHONY: mock

migrate-create:  ### create new migration
	migrate create -ext sql -dir /internal/app/migrations 'migrate_name'
.PHONY: migrate-create

migrate-up: ### migration up
	migrate -path /internal/app/migrations -database '$(DB_URL)?sslmode=disable' up
.PHONY: migrate-up

bin-deps:
	GOBIN=$(LOCAL_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	GOBIN=$(LOCAL_BIN) go install go.uber.org/mock/mockgen@latest

##@ Redfish API Generation
# Redfish API Generation Variables
MAIN_OPENAPI := doc/redfish-api/openapi.yaml
MERGED_OPENAPI := doc/redfish-openapi.yaml
GENERATED_SPEC := internal/controller/http/redfish/v1/redfishapi/spec.gen.go
GENERATED_TYPES := internal/controller/http/redfish/v1/redfishapi/types.gen.go
GENERATED_SERVER := internal/controller/http/redfish/v1/redfishapi/server.gen.go
MERGE_SCRIPT := merge-redfish-openapi.py

# Source YAML files in doc/redfish-api directory
YAML_SOURCES := $(wildcard doc/redfish-api/*.yaml)

# Required tools for Redfish API generation
REDFISH_REQUIRED_TOOLS := python3 go oapi-codegen
REDFISH_OPTIONAL_TOOLS := jq curl swagger-cli
PYTHON_TOOLS := PyYAML

# Tool check functions
define check_tool
$(shell command -v $(1) >/dev/null 2>&1 && echo "✅" || echo "❌")
endef

define check_python_package
$(shell python3 -c "import $(1)" 2>/dev/null && echo "✅" || echo "❌")
endef

# OS check function for Ubuntu
define check_ubuntu
	@if [ ! -f /etc/os-release ]; then \
		echo "❌ Cannot detect OS. /etc/os-release not found."; \
		echo "💡 Redfish API targets are only supported on Ubuntu systems."; \
		exit 1; \
	fi; \
	if ! grep -qi "ubuntu" /etc/os-release; then \
		echo "❌ Redfish API targets are only supported on Ubuntu systems."; \
		echo "🔍 Current OS: $$(grep '^PRETTY_NAME=' /etc/os-release | cut -d'=' -f2 | tr -d '\"' || echo 'Unknown')"; \
		echo "💡 Please use an Ubuntu system to run Redfish API generation targets."; \
		exit 1; \
	fi; \
	echo "✅ Ubuntu detected: $$(grep '^PRETTY_NAME=' /etc/os-release | cut -d'=' -f2 | tr -d '\"')"
endef

rf-merge: ### Merge YAML files into single OpenAPI specification
	$(call check_ubuntu)
	@echo "🔄 Merging YAML files into single OpenAPI specification..."
	python3 $(MERGE_SCRIPT)
	@echo "✅ Merged OpenAPI spec created: $(MERGED_OPENAPI)"
.PHONY: rf-merge

$(MERGED_OPENAPI): $(YAML_SOURCES) $(MERGE_SCRIPT)
	@echo "🔄 Merging YAML files into single OpenAPI specification..."
	python3 $(MERGE_SCRIPT)
	@echo "✅ Merged OpenAPI spec created: $(MERGED_OPENAPI)"

rf-generate: $(GENERATED_SPEC) $(GENERATED_TYPES) $(GENERATED_SERVER) ### Generate Go server code from merged OpenAPI spec
	$(call check_ubuntu)
.PHONY: rf-generate

$(GENERATED_SPEC): $(MERGED_OPENAPI)
	$(call check_ubuntu)
	@echo "🔄 Generating OpenAPI spec Go code..."
	@mkdir -p internal/controller/http/redfish/v1/redfishapi
	oapi-codegen -generate spec -o $(GENERATED_SPEC) -package redfishapi $(MERGED_OPENAPI)
	@echo "✅ Generated spec code: $(GENERATED_SPEC)"

$(GENERATED_TYPES): $(MERGED_OPENAPI)
	$(call check_ubuntu)
	@echo "🔄 Generating types Go code..."
	@mkdir -p internal/controller/http/redfish/v1/redfishapi
	oapi-codegen -generate types -o $(GENERATED_TYPES) -package redfishapi $(MERGED_OPENAPI)
	@echo "✅ Generated types code: $(GENERATED_TYPES)"

$(GENERATED_SERVER): $(MERGED_OPENAPI)
	$(call check_ubuntu)
	@echo "🔄 Generating server Go code..."
	@mkdir -p internal/controller/http/redfish/v1/redfishapi
	oapi-codegen -generate gin,strict-server -o $(GENERATED_SERVER) -package redfishapi $(MERGED_OPENAPI)
	@echo "✅ Generated server code: $(GENERATED_SERVER)"

rf-deps: rf-check-tools ### Install dependencies and required tools for Redfish API
	$(call check_ubuntu)
	@echo "🔧 Installing and verifying all dependencies..."
	@$(MAKE) rf-install-missing-tools
	@echo "✅ All dependencies installed and verified"
.PHONY: rf-deps

rf-check-tools: ### Check if all required tools are available
	$(call check_ubuntu)
	@echo "🔧 Detailed Tool Status Report"
	@echo "==============================="
	@echo ""
	@echo "Required Tools:"
	@missing_tools=""; \
	for tool in $(REDFISH_REQUIRED_TOOLS); do \
		echo -n "  $$tool: "; \
		if command -v $$tool >/dev/null 2>&1; then \
			case $$tool in \
				python3) echo "✅ $$(python3 --version 2>&1)";; \
				go) echo "✅ $$(go version | cut -d' ' -f3-4)";; \
				oapi-codegen) echo "✅ $$(oapi-codegen -version 2>/dev/null || echo 'installed')";; \
				*) echo "✅ installed";; \
			esac; \
		else \
			echo "❌ Missing"; \
			missing_tools="$$missing_tools $$tool"; \
		fi; \
	done; \
	echo ""; \
	echo "Optional Tools:"; \
	for tool in $(REDFISH_OPTIONAL_TOOLS); do \
		echo -n "  $$tool: "; \
		if command -v $$tool >/dev/null 2>&1; then \
			case $$tool in \
				jq) echo "✅ $$(jq --version 2>/dev/null || echo 'installed')";; \
				curl) echo "✅ $$(curl --version 2>/dev/null | head -n1 | cut -d' ' -f2 || echo 'installed')";; \
				swagger-cli) echo "✅ $$(swagger-cli --version 2>/dev/null || echo 'installed')";; \
				*) echo "✅ installed";; \
			esac; \
		else \
			echo "⚠️  Not installed"; \
		fi; \
	done; \
	echo ""; \
	echo "Python Packages:"; \
	echo -n "  PyYAML: "; \
	if python3 -c "import yaml; print('✅ version', yaml.__version__)" 2>/dev/null; then \
		true; \
	else \
		echo "❌ Missing"; \
	fi; \
	echo ""; \
	if [ -n "$$missing_tools" ]; then \
		echo "❌ Missing required tools:$$missing_tools"; \
		echo "💡 Run 'make rf-deps' to install missing dependencies"; \
		exit 1; \
	else \
		echo "✅ All required tools are available!"; \
	fi
.PHONY: rf-check-tools

rf-install-missing-tools: ### Install missing tools automatically
	$(call check_ubuntu)
	@echo "🛠️  Installing missing tools..."

	# Check and install Go
	@if ! command -v go >/dev/null 2>&1; then \
		echo "❌ Go not found. Please install Go first: https://golang.org/doc/install"; \
		exit 1; \
	fi

	# Check and install Python3
	@if ! command -v python3 >/dev/null 2>&1; then \
		echo "Installing Python3..."; \
		if command -v apt-get >/dev/null 2>&1; then \
			sudo apt-get update && sudo apt-get install -y python3 python3-pip; \
		elif command -v yum >/dev/null 2>&1; then \
			sudo yum install -y python3 python3-pip; \
		else \
			echo "❌ Please install Python3 manually"; \
			exit 1; \
		fi; \
	fi

	# Install Python packages
	@if ! python3 -c "import yaml" 2>/dev/null; then \
		echo "Installing PyYAML..."; \
		pip3 install PyYAML --user 2>/dev/null || python3 -m pip install PyYAML --user; \
	fi

	# Install oapi-codegen
	@if ! command -v oapi-codegen >/dev/null 2>&1; then \
		echo "Installing oapi-codegen..."; \
		go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest; \
	fi

	# Install system tools based on OS
	@if command -v apt-get >/dev/null 2>&1; then \
		echo "Installing system tools (Debian/Ubuntu)..."; \
		sudo apt-get update >/dev/null 2>&1 || true; \
		sudo apt-get install -y jq curl >/dev/null 2>&1 || echo "⚠️  Some tools may need manual installation"; \
	elif command -v yum >/dev/null 2>&1; then \
		echo "Installing system tools (RHEL/CentOS)..."; \
		sudo yum install -y jq curl >/dev/null 2>&1 || echo "⚠️  Some tools may need manual installation"; \
	elif command -v brew >/dev/null 2>&1; then \
		echo "Installing system tools (macOS)..."; \
		brew install jq curl 2>/dev/null || echo "⚠️  Some tools may need manual installation"; \
	else \
		echo "⚠️  Unknown package manager - please install jq and curl manually"; \
	fi
.PHONY: rf-install-missing-tools

rf-validate: $(MERGED_OPENAPI) ### Validate OpenAPI specification
	$(call check_ubuntu)
	@echo "🔄 Validating OpenAPI specification..."
	@if command -v swagger-cli >/dev/null 2>&1; then \
		swagger-cli validate $(MERGED_OPENAPI) && echo "✅ OpenAPI spec is valid"; \
	else \
		echo "⚠️  swagger-cli not found, skipping validation"; \
		echo "💡 Install swagger-cli: npm install -g @apidevtools/swagger-cli"; \
	fi
.PHONY: rf-validate
