TUTORIALS := 01_basic 02_streaming 03_tool_use 04_thinking 05_error_handling \
	06_cost_and_registry 07_logging_and_middleware 08_auth_pool_failover \
	09_system_and_multiturn 10_streaming_tool_use 11_stream_abort_and_errors \
	12_request_overrides 13_registry_deep 14_provider_helpers 15_multi_provider \
	16_pool_deep_dive 17_error_constructors 18_content_model

TEST_SUITES := 19_stress_tests 20_capability_test 21_cloud_ctl_tool_use

ALL_MODULES := . $(TUTORIALS) $(TEST_SUITES)

.PHONY: build-all test-stress test-capability test-all vet-all tidy-all update-deps clean

## build-all: compile all tutorial binaries (01–18)
build-all:
	@for dir in $(TUTORIALS); do \
		echo "==> Building $$dir"; \
		(cd $$dir && go build -o /dev/null .); \
	done
	@echo "All tutorials compile OK"

## test-stress: run tutorial 19 stress tests with race detector
test-stress:
	cd 19_stress_tests && go test -race -count=1 ./...

## test-capability: run tutorial 20 capability tests (requires API keys)
test-capability:
	cd 20_capability_test && go test -count=1 -timeout 120m ./...

## test-all: compile tutorials + run all test suites
test-all: build-all test-stress
	@echo "All tests passed"

## vet-all: run go vet on every module
vet-all:
	@for dir in $(ALL_MODULES); do \
		echo "==> Vetting $$dir"; \
		(cd $$dir && go vet ./...); \
	done
	@echo "All modules vet OK"

## tidy-all: run go mod tidy on every module
tidy-all:
	@for dir in $(ALL_MODULES); do \
		echo "==> Tidying $$dir"; \
		(cd $$dir && go mod tidy); \
	done
	@echo "All modules tidied"

## update-deps: update rho/llm to latest across all modules
update-deps:
	@for dir in $(ALL_MODULES); do \
		echo "==> Updating $$dir"; \
		(cd $$dir && go get github.com/bds421/rho-llm@latest && go mod tidy); \
	done
	@echo "All modules updated"

## clean: remove compiled binaries
clean:
	@for dir in $(TUTORIALS); do \
		rm -f $$dir/$$(basename $$dir); \
	done
	@echo "Cleaned"
