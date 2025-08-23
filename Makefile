# BindDiff Makefile
# 提供便捷的构建、测试和部署命令

# 变量定义
BINARY_NAME = bdiff
VERSION = 2.0.0
BUILD_DIR = build
DIST_DIR = dist
GO_VERSION = 1.21

# Go 相关变量
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# 构建标志
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

# 默认目标
.PHONY: all
all: clean deps test build

# 安装依赖
.PHONY: deps
deps:
	@echo "📦 Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 清理
.PHONY: clean
clean:
	@echo "🧹 Cleaning up..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f $(BINARY_NAME)
	rm -f benchmark
	rm -f *.prof
	rm -f *.log

# 构建
.PHONY: build
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

# 构建基准测试工具
.PHONY: build-benchmark
build-benchmark:
	@echo "🔨 Building benchmark tool..."
	$(GOBUILD) -o benchmark ./benchmark/

# 快速构建（开发用）
.PHONY: dev
dev:
	@echo "⚡ Quick development build..."
	$(GOBUILD) -o $(BINARY_NAME) ./main.go

# 运行测试
.PHONY: test
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 生成完整测试报告 (HTML格式)
.PHONY: test-report
test-report:
	@echo "📋 Generating test report..."
	@if [ -f "scripts/test-report.sh" ]; then \
		chmod +x scripts/test-report.sh && \
		scripts/test-report.sh -f html -c; \
	else \
		echo "Test report script not found. Please ensure scripts/test-report.sh exists."; \
	fi

# 生成完整测试报告 (所有格式)
.PHONY: test-report-all
test-report-all:
	@echo "📋 Generating comprehensive test report..."
	@if [ -f "scripts/test-report.sh" ]; then \
		chmod +x scripts/test-report.sh && \
		scripts/test-report.sh -f all -c -b -p; \
	else \
		echo "Test report script not found. Please ensure scripts/test-report.sh exists."; \
	fi

# 快速测试报告 (仅测试结果)
.PHONY: test-report-quick
test-report-quick:
	@echo "⚡ Generating quick test report..."
	@if [ -f "scripts/test-report.sh" ]; then \
		chmod +x scripts/test-report.sh && \
		scripts/test-report.sh -f html; \
	else \
		echo "Test report script not found. Please ensure scripts/test-report.sh exists."; \
	fi

# 运行基准测试
.PHONY: bench
bench:
	@echo "⚡ Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./core/

# 运行完整性能测试
.PHONY: benchmark
benchmark: build-benchmark
	@echo "📈 Running performance tests..."
	./benchmark test

# 代码格式化
.PHONY: fmt
fmt:
	@echo "💅 Formatting code..."
	$(GOCMD) fmt ./...

# 代码静态检查
.PHONY: vet
vet:
	@echo "🔍 Vetting code..."
	$(GOCMD) vet ./...

# Lint 检查 (需要 golangci-lint)
.PHONY: lint
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping lint check"; \
		echo "Install it with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2"; \
	fi

# 代码质量检查（格式化 + 静态检查 + Lint）
.PHONY: check
check: fmt vet lint

# 性能分析
.PHONY: profile
profile: build-benchmark
	@echo "🎯 Running CPU profiling..."
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./core/
	@echo "Profile generated: cpu.prof"
	@echo "View with: go tool pprof cpu.prof"

# 内存分析
.PHONY: profile-mem
profile-mem: build-benchmark
	@echo "🧠 Running memory profiling..."
	$(GOTEST) -memprofile=mem.prof -bench=. ./core/
	@echo "Profile generated: mem.prof"
	@echo "View with: go tool pprof mem.prof"

# 跨平台构建
.PHONY: build-all
build-all: clean
	@echo "🌍 Building for all platforms..."
	mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./main.go
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./main.go
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./main.go
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./main.go
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./main.go
	
	@echo "✅ All builds completed in $(DIST_DIR)/"

# 创建发布包
.PHONY: release
release: build-all
	@echo "📦 Creating release packages..."
	cd $(DIST_DIR) && \
	for binary in $(BINARY_NAME)-*; do \
		if [[ $$binary == *".exe" ]]; then \
			zip $${binary%.*}.zip $$binary ../README.md ../bindiff.yaml; \
		else \
			tar -czf $$binary.tar.gz $$binary ../README.md ../bindiff.yaml; \
		fi \
	done
	@echo "✅ Release packages created in $(DIST_DIR)/"

# 安装到系统
.PHONY: install
install: build
	@echo "📍 Installing $(BINARY_NAME)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Uninstalled"

# 运行示例
.PHONY: example
example: build
	@echo "🎯 Running example..."
	mkdir -p test_files
	echo "Hello, World! This is version 1." > test_files/file1.txt
	echo "Hello, World! This is version 2 with changes." > test_files/file2.txt
	
	@echo "Creating diff..."
	$(BUILD_DIR)/$(BINARY_NAME) diff test_files/file1.txt test_files/file2.txt -o test_files/patch.bdf
	
	@echo "Applying patch..."
	$(BUILD_DIR)/$(BINARY_NAME) apply test_files/file1.txt test_files/patch.bdf -o test_files/result.txt
	
	@echo "Verifying result..."
	diff test_files/file2.txt test_files/result.txt && echo "✅ Example completed successfully!" || echo "❌ Example failed!"
	
	rm -rf test_files

# Docker 构建
.PHONY: docker-build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t bindiff:$(VERSION) .
	docker tag bindiff:$(VERSION) bindiff:latest

# 生成文档
.PHONY: docs
docs:
	@echo "📚 Generating documentation..."
	$(GOCMD) doc -all ./... > docs/api.md
	@echo "✅ Documentation generated in docs/"

# 初始化开发环境
.PHONY: dev-setup
dev-setup:
	@echo "🛠️  Setting up development environment..."
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ Development environment ready"

# 显示帮助信息
.PHONY: help
help:
	@echo "BindDiff Makefile Commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms" 
	@echo "  dev            - Quick development build"
	@echo "  release        - Create release packages"
	@echo ""
	@echo "Test Commands:"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-report    - Generate HTML test report with coverage"
	@echo "  test-report-all - Generate comprehensive test report (all formats)"
	@echo "  test-report-quick - Generate quick HTML test report"
	@echo "  bench          - Run benchmarks"
	@echo "  benchmark      - Run performance tests"
	@echo ""
	@echo "Quality Commands:"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run linter"
	@echo "  check          - Run all quality checks"
	@echo ""
	@echo "Profile Commands:"
	@echo "  profile        - CPU profiling"
	@echo "  profile-mem    - Memory profiling"
	@echo ""
	@echo "Utility Commands:"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  install        - Install to system"
	@echo "  uninstall      - Uninstall from system"
	@echo "  example        - Run example"
	@echo "  help           - Show this help"