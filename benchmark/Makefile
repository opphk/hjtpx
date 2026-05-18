.PHONY: benchmark benchmark-quick benchmark-full benchmark-baseline benchmark-compare benchmark-ci benchmark-serve benchmark-vegeta benchmark-wrk benchmark-all help

help:
	@echo "Performance Benchmark Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  benchmark            Run standard benchmarks (60s duration)"
	@echo "  benchmark-quick      Run quick benchmarks (30s duration)"
	@echo "  benchmark-full      Run full benchmarks (120s duration)"
	@echo "  benchmark-baseline  Generate and save baseline"
	@echo "  benchmark-compare   Compare with existing baseline"
	@echo "  benchmark-ci        Run in CI mode"
	@echo "  benchmark-serve     Start service and run benchmarks"
	@echo "  benchmark-reports   Generate reports from existing results"
	@echo "  benchmark-clean     Clean benchmark reports and profiles"
	@echo "  benchmark-vegeta    Run Vegeta-based benchmarks"
	@echo "  benchmark-wrk       Run wrk-based benchmarks"
	@echo "  benchmark-all       Run all benchmark tools"

BENCHMARK_DIR := $(shell pwd)
REPORT_DIR := $(BENCHMARK_DIR)/reports
BASELINE_DIR := $(BENCHMARK_DIR)/baselines

benchmark:
	@echo "Running standard benchmarks..."
	@mkdir -p $(REPORT_DIR)
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--duration=60 \
		--concurrency=100 \
		--format=both \
		--output=$(REPORT_DIR)

benchmark-quick:
	@echo "Running quick benchmarks..."
	@mkdir -p $(REPORT_DIR)
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--duration=30 \
		--concurrency=50 \
		--format=both \
		--output=$(REPORT_DIR)

benchmark-full:
	@echo "Running full benchmarks..."
	@mkdir -p $(REPORT_DIR)
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--duration=120 \
		--concurrency=200 \
		--format=both \
		--output=$(REPORT_DIR)

benchmark-baseline:
	@echo "Generating baseline..."
	@mkdir -p $(BASELINE_DIR) $(REPORT_DIR)
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--duration=120 \
		--concurrency=200 \
		--baseline \
		--format=both \
		--output=$(REPORT_DIR)

benchmark-compare:
	@echo "Comparing with baseline..."
	@mkdir -p $(REPORT_DIR)
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--compare \
		--format=both \
		--output=$(REPORT_DIR)

benchmark-ci:
	@echo "Running CI benchmark..."
	@mkdir -p $(REPORT_DIR) $(BASELINE_DIR)
	@export CI=true && cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--duration=60 \
		--concurrency=100 \
		--compare \
		--format=json \
		--no-color \
		--output=$(REPORT_DIR)

benchmark-serve:
	@echo "Starting service and running benchmarks..."
	@mkdir -p $(REPORT_DIR)
	@cd $(BENCHMARK_DIR)/../backend && \
		nohup go run cmd/api/main.go > ../backend.log 2>&1 & \
		echo $$! > ../backend.pid && \
		echo "Backend started (PID: $$(cat ../backend.pid))"
	@echo "Waiting for service to be ready..."
	@for i in $$(seq 1 60); do \
		if curl -sf http://localhost:8080/health > /dev/null 2>&1; then \
			echo "Service is ready"; \
			break; \
		fi; \
		echo -n "."; \
		sleep 2; \
	done
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--duration=60 \
		--concurrency=100 \
		--format=both \
		--output=$(REPORT_DIR)
	@echo "Stopping service..."
	@cd $(BENCHMARK_DIR)/../backend && \
		if [ -f ../backend.pid ]; then \
			kill $$(cat ../backend.pid) 2>/dev/null || true; \
			rm -f ../backend.pid; \
		fi

benchmark-reports:
	@echo "Generating reports..."
	@mkdir -p $(REPORT_DIR)
	@cd $(BENCHMARK_DIR) && go run cmd/benchmark/main.go \
		--format=both \
		--output=$(REPORT_DIR) \
		--quiet

benchmark-clean:
	@echo "Cleaning benchmark data..."
	@rm -rf $(REPORT_DIR)
	@rm -f $(BENCHMARK_DIR)/*.prof
	@rm -f $(BENCHMARK_DIR)/benchmark_report_*.json
	@rm -f $(BENCHMARK_DIR)/benchmark_report_*.html
	@echo "Cleaned."

benchmark-test:
	@echo "Running benchmark unit tests..."
	@cd $(BENCHMARK_DIR) && go test -v -bench=. -benchmem -run=^$$

benchmark-vegeta:
	@echo "Running Vegeta benchmarks..."
	@chmod +x $(BENCHMARK_DIR)/vegeta.sh
	@$(BENCHMARK_DIR)/vegeta.sh all

benchmark-vegeta-install:
	@echo "Installing Vegeta..."
	@chmod +x $(BENCHMARK_DIR)/vegeta.sh
	@$(BENCHMARK_DIR)/vegeta.sh install

benchmark-wrk:
	@echo "Running wrk benchmarks..."
	@chmod +x $(BENCHMARK_DIR)/wrk.sh
	@$(BENCHMARK_DIR)/wrk.sh all

benchmark-wrk-install:
	@echo "Installing wrk..."
	@chmod +x $(BENCHMARK_DIR)/wrk.sh
	@$(BENCHMARK_DIR)/wrk.sh install

benchmark-all:
	@echo "Running all benchmark tools..."
	@$(MAKE) benchmark
	@$(MAKE) benchmark-vegeta
	@$(MAKE) benchmark-wrk
