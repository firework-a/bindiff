#!/bin/bash

# BindDiff 测试报告生成脚本
# 支持生成 HTML、JSON、XML 格式的测试报告

set -e

# 设置UTF-8编码
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORT_DIR="$PROJECT_ROOT/test-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
BindDiff 测试报告生成器

用法: $0 [选项]

选项:
    -h, --help          显示此帮助信息
    -f, --format        报告格式 (html|json|xml|all) [默认: html]
    -o, --output        输出目录 [默认: test-reports]
    -v, --verbose       详细输出
    -c, --coverage      生成覆盖率报告
    -b, --benchmark     运行基准测试
    -p, --profile       生成性能分析报告
    --no-cleanup        不清理临时文件

示例:
    $0                           # 生成基本HTML报告
    $0 -f all -c -b             # 生成所有格式报告，包含覆盖率和基准测试
    $0 -f json -o custom-dir    # 生成JSON格式报告到自定义目录
EOF
}

# 解析命令行参数
FORMAT="html"
OUTPUT_DIR=""
VERBOSE=false
COVERAGE=false
BENCHMARK=false
PROFILE=false
NO_CLEANUP=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -f|--format)
            FORMAT="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -b|--benchmark)
            BENCHMARK=true
            shift
            ;;
        -p|--profile)
            PROFILE=true
            shift
            ;;
        --no-cleanup)
            NO_CLEANUP=true
            shift
            ;;
        *)
            log_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
done

# 设置输出目录
if [[ -z "$OUTPUT_DIR" ]]; then
    OUTPUT_DIR="$REPORT_DIR"
fi

# 验证格式
if [[ "$FORMAT" != "html" && "$FORMAT" != "json" && "$FORMAT" != "xml" && "$FORMAT" != "all" ]]; then
    log_error "不支持的格式: $FORMAT"
    show_help
    exit 1
fi

# 创建必要目录
mkdir -p "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR/coverage"
mkdir -p "$OUTPUT_DIR/benchmark"
mkdir -p "$OUTPUT_DIR/profile"

# 进入项目根目录
cd "$PROJECT_ROOT"

log_info "开始生成测试报告..."
log_info "项目根目录: $PROJECT_ROOT"
log_info "输出目录: $OUTPUT_DIR"
log_info "报告格式: $FORMAT"

# 检查必要工具
check_tools() {
    local tools=("go")
    
    if [[ "$COVERAGE" == "true" ]]; then
        tools+=("gcov2lcov" "genhtml")
    fi
    
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            log_warning "工具 $tool 未找到，某些功能可能不可用"
        fi
    done
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    local test_output="$OUTPUT_DIR/test-results.txt"
    local json_output="$OUTPUT_DIR/test-results.json"
    
    # 运行测试并保存输出
    if [[ "$VERBOSE" == "true" ]]; then
        go test -v ./test/... | tee "$test_output"
    else
        go test ./test/... | tee "$test_output"
    fi
    
    local test_exit_code=$?
    
    # 生成JSON格式的测试结果
    go test -json ./test/... > "$json_output" 2>/dev/null || true
    
    if [[ $test_exit_code -eq 0 ]]; then
        log_success "单元测试通过"
    else
        log_warning "单元测试有失败项"
    fi
    
    return $test_exit_code
}

# 生成覆盖率报告
generate_coverage() {
    if [[ "$COVERAGE" != "true" ]]; then
        return 0
    fi
    
    log_info "生成覆盖率报告..."
    
    local coverage_out="$OUTPUT_DIR/coverage/coverage.out"
    local coverage_html="$OUTPUT_DIR/coverage/coverage.html"
    local coverage_json="$OUTPUT_DIR/coverage/coverage.json"
    
    # 运行覆盖率测试
    go test -race -coverprofile="$coverage_out" -covermode=atomic ./test/...
    
    if [[ -f "$coverage_out" ]]; then
        # 生成HTML报告
        go tool cover -html="$coverage_out" -o "$coverage_html"
        log_success "覆盖率HTML报告生成: $coverage_html"
        
        # 生成JSON格式覆盖率数据
        go tool cover -func="$coverage_out" | grep "^total:" | awk '{print "{\"total_coverage\": \"" $3 "\"}"}' > "$coverage_json"
        
        # 显示覆盖率摘要
        local total_coverage=$(go tool cover -func="$coverage_out" | grep "^total:" | awk '{print $3}')
        log_info "总体代码覆盖率: $total_coverage"
    else
        log_warning "覆盖率数据文件未生成"
    fi
}

# 运行基准测试
run_benchmark() {
    if [[ "$BENCHMARK" != "true" ]]; then
        return 0
    fi
    
    log_info "运行基准测试..."
    
    local bench_output="$OUTPUT_DIR/benchmark/benchmark-results.txt"
    local bench_json="$OUTPUT_DIR/benchmark/benchmark-results.json"
    
    # 运行基准测试
    go test -bench=. -benchmem ./test/... | tee "$bench_output"
    
    # 尝试生成JSON格式（如果支持）
    go test -bench=. -benchmem -json ./test/... > "$bench_json" 2>/dev/null || true
    
    log_success "基准测试报告生成: $bench_output"
}

# 生成性能分析报告
generate_profile() {
    if [[ "$PROFILE" != "true" ]]; then
        return 0
    fi
    
    log_info "生成性能分析报告..."
    
    local cpu_prof="$OUTPUT_DIR/profile/cpu.prof"
    local mem_prof="$OUTPUT_DIR/profile/mem.prof"
    
    # CPU性能分析
    go test -cpuprofile="$cpu_prof" -bench=. ./test/core/ >/dev/null 2>&1 || true
    if [[ -f "$cpu_prof" ]]; then
        log_success "CPU性能分析文件生成: $cpu_prof"
        log_info "查看CPU分析: go tool pprof $cpu_prof"
    fi
    
    # 内存性能分析
    go test -memprofile="$mem_prof" -bench=. ./test/core/ >/dev/null 2>&1 || true
    if [[ -f "$mem_prof" ]]; then
        log_success "内存性能分析文件生成: $mem_prof"
        log_info "查看内存分析: go tool pprof $mem_prof"
    fi
}

# 生成HTML报告
generate_html_report() {
    log_info "生成HTML测试报告..."
    
    local html_file="$OUTPUT_DIR/test-report-$TIMESTAMP.html"
    
    cat > "$html_file" << 'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BindDiff 测试报告</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 2rem; border-radius: 10px; margin-bottom: 2rem; text-align: center; }
        .header h1 { font-size: 2.5rem; margin-bottom: 0.5rem; }
        .header p { font-size: 1.1rem; opacity: 0.9; }
        .section { background: white; margin-bottom: 2rem; padding: 1.5rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .section h2 { color: #333; margin-bottom: 1rem; padding-bottom: 0.5rem; border-bottom: 3px solid #667eea; }
        .status-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1rem; margin: 1rem 0; }
        .status-card { padding: 1rem; border-radius: 6px; text-align: center; }
        .status-pass { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .status-fail { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
        .status-warning { background: #fff3cd; border: 1px solid #ffeaa7; color: #856404; }
        .metric { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid #eee; }
        .metric:last-child { border-bottom: none; }
        .metric-label { font-weight: 600; }
        .metric-value { color: #666; }
        pre { background: #f8f9fa; padding: 1rem; border-radius: 4px; overflow-x: auto; margin: 1rem 0; border: 1px solid #dee2e6; }
        .footer { text-align: center; padding: 2rem; color: #666; }
        .tab-container { margin: 1rem 0; }
        .tab-buttons { display: flex; border-bottom: 1px solid #ddd; }
        .tab-button { padding: 0.75rem 1.5rem; background: #f8f9fa; border: none; cursor: pointer; border-bottom: 3px solid transparent; }
        .tab-button.active { background: white; border-bottom-color: #667eea; color: #667eea; font-weight: 600; }
        .tab-content { display: none; padding: 1rem 0; }
        .tab-content.active { display: block; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>BindDiff 测试报告</h1>
            <p>生成时间: __TIMESTAMP__</p>
        </div>
        
        <div class="section">
            <h2>📊 测试概要</h2>
            <div class="status-grid">
                <div class="status-card __TEST_STATUS_CLASS__">
                    <h3>单元测试</h3>
                    <p><strong>__TEST_RESULT__</strong></p>
                </div>
                <div class="status-card __COVERAGE_STATUS_CLASS__">
                    <h3>代码覆盖率</h3>
                    <p><strong>__COVERAGE_RESULT__</strong></p>
                </div>
                <div class="status-card __BENCHMARK_STATUS_CLASS__">
                    <h3>基准测试</h3>
                    <p><strong>__BENCHMARK_RESULT__</strong></p>
                </div>
            </div>
        </div>
        
        <div class="section">
            <h2>📈 关键指标</h2>
            <div class="metric">
                <span class="metric-label">测试用例总数:</span>
                <span class="metric-value">__TOTAL_TESTS__</span>
            </div>
            <div class="metric">
                <span class="metric-label">通过测试:</span>
                <span class="metric-value">__PASSED_TESTS__</span>
            </div>
            <div class="metric">
                <span class="metric-label">失败测试:</span>
                <span class="metric-value">__FAILED_TESTS__</span>
            </div>
            <div class="metric">
                <span class="metric-label">代码覆盖率:</span>
                <span class="metric-value">__TOTAL_COVERAGE__</span>
            </div>
            <div class="metric">
                <span class="metric-label">测试执行时间:</span>
                <span class="metric-value">__EXECUTION_TIME__</span>
            </div>
        </div>
        
        <div class="section">
            <h2>📝 详细结果</h2>
            <div class="tab-container">
                <div class="tab-buttons">
                    <button class="tab-button active" onclick="showTab('unit-tests')">单元测试</button>
                    <button class="tab-button" onclick="showTab('coverage')" style="display: __COVERAGE_DISPLAY__">覆盖率</button>
                    <button class="tab-button" onclick="showTab('benchmark')" style="display: __BENCHMARK_DISPLAY__">基准测试</button>
                </div>
                
                <div class="tab-content active" id="unit-tests">
                    <h3>单元测试结果</h3>
                    <pre>__TEST_OUTPUT__</pre>
                </div>
                
                <div class="tab-content" id="coverage" style="display: __COVERAGE_DISPLAY__">
                    <h3>覆盖率报告</h3>
                    <p>详细覆盖率报告: <a href="coverage/coverage.html" target="_blank">查看HTML报告</a></p>
                    <pre>__COVERAGE_OUTPUT__</pre>
                </div>
                
                <div class="tab-content" id="benchmark" style="display: __BENCHMARK_DISPLAY__">
                    <h3>基准测试结果</h3>
                    <pre>__BENCHMARK_OUTPUT__</pre>
                </div>
            </div>
        </div>
        
        <div class="footer">
            <p>BindDiff v2.0.0 - 高性能二进制差异分析工具</p>
            <p>报告生成器 by BindDiff Team</p>
        </div>
    </div>
    
    <script>
        function showTab(tabName) {
            // 隐藏所有标签内容
            const contents = document.querySelectorAll('.tab-content');
            contents.forEach(content => content.classList.remove('active'));
            
            // 移除所有按钮的活动状态
            const buttons = document.querySelectorAll('.tab-button');
            buttons.forEach(button => button.classList.remove('active'));
            
            // 显示选中的标签内容
            document.getElementById(tabName).classList.add('active');
            event.target.classList.add('active');
        }
    </script>
</body>
</html>
EOF

    # 读取测试结果并填充模板
    local test_output=""
    local coverage_output=""
    local benchmark_output=""
    local test_status="通过"
    local test_status_class="status-pass"
    
    # 读取测试输出
    if [[ -f "$OUTPUT_DIR/test-results.txt" ]]; then
        test_output=$(cat "$OUTPUT_DIR/test-results.txt" | head -50)  # 限制行数
    else
        test_output="测试输出文件未找到"
        test_status="失败"
        test_status_class="status-fail"
    fi
    
    # 读取覆盖率输出
    local coverage_display="none"
    local coverage_status_class="status-warning"
    local coverage_result="未运行"
    if [[ "$COVERAGE" == "true" && -f "$OUTPUT_DIR/coverage/coverage.out" ]]; then
        coverage_display="block"
        coverage_output=$(go tool cover -func="$OUTPUT_DIR/coverage/coverage.out" | tail -10)
        coverage_result=$(go tool cover -func="$OUTPUT_DIR/coverage/coverage.out" | grep "^total:" | awk '{print $3}' || echo "N/A")
        coverage_status_class="status-pass"
    fi
    
    # 读取基准测试输出
    local benchmark_display="none"
    local benchmark_status_class="status-warning"
    local benchmark_result="未运行"
    if [[ "$BENCHMARK" == "true" && -f "$OUTPUT_DIR/benchmark/benchmark-results.txt" ]]; then
        benchmark_display="block"
        benchmark_output=$(cat "$OUTPUT_DIR/benchmark/benchmark-results.txt" | head -30)
        benchmark_result="已完成"
        benchmark_status_class="status-pass"
    fi
    
    # 计算测试统计
    local total_tests=$(echo "$test_output" | grep -c "PASS\|FAIL" || echo "0")
    local passed_tests=$(echo "$test_output" | grep -c "PASS" || echo "0")
    local failed_tests=$(echo "$test_output" | grep -c "FAIL" || echo "0")
    
    # 替换模板变量
    sed -i "s|__TIMESTAMP__|$(date '+%Y-%m-%d %H:%M:%S')|g" "$html_file"
    sed -i "s|__TEST_STATUS_CLASS__|$test_status_class|g" "$html_file"
    sed -i "s|__TEST_RESULT__|$test_status|g" "$html_file"
    sed -i "s|__COVERAGE_STATUS_CLASS__|$coverage_status_class|g" "$html_file"
    sed -i "s|__COVERAGE_RESULT__|$coverage_result|g" "$html_file"
    sed -i "s|__BENCHMARK_STATUS_CLASS__|$benchmark_status_class|g" "$html_file"
    sed -i "s|__BENCHMARK_RESULT__|$benchmark_result|g" "$html_file"
    sed -i "s|__TOTAL_TESTS__|$total_tests|g" "$html_file"
    sed -i "s|__PASSED_TESTS__|$passed_tests|g" "$html_file"
    sed -i "s|__FAILED_TESTS__|$failed_tests|g" "$html_file"
    sed -i "s|__TOTAL_COVERAGE__|$coverage_result|g" "$html_file"
    sed -i "s|__EXECUTION_TIME__|$(date '+%Y-%m-%d %H:%M:%S')|g" "$html_file"
    sed -i "s|__COVERAGE_DISPLAY__|$coverage_display|g" "$html_file"
    sed -i "s|__BENCHMARK_DISPLAY__|$benchmark_display|g" "$html_file"
    
    # 处理多行内容
    local escaped_test_output=$(echo "$test_output" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
    local escaped_coverage_output=$(echo "$coverage_output" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
    local escaped_benchmark_output=$(echo "$benchmark_output" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
    
    # 使用临时文件处理复杂替换
    sed "s|__TEST_OUTPUT__|${escaped_test_output}|g" "$html_file" > "$html_file.tmp"
    sed "s|__COVERAGE_OUTPUT__|${escaped_coverage_output}|g" "$html_file.tmp" > "$html_file.tmp2"
    sed "s|__BENCHMARK_OUTPUT__|${escaped_benchmark_output}|g" "$html_file.tmp2" > "$html_file"
    rm -f "$html_file.tmp" "$html_file.tmp2"
    
    log_success "HTML测试报告生成: $html_file"
}

# 生成JSON报告
generate_json_report() {
    log_info "生成JSON测试报告..."
    
    local json_file="$OUTPUT_DIR/test-report-$TIMESTAMP.json"
    
    # 构建JSON报告
    cat > "$json_file" << EOF
{
    "report": {
        "generated_at": "$(date -Iseconds)",
        "project": "BindDiff",
        "version": "2.0.0",
        "test_framework": "go test"
    },
    "summary": {
        "total_tests": 0,
        "passed_tests": 0,
        "failed_tests": 0,
        "coverage_percentage": "0%",
        "execution_time_seconds": 0
    },
    "results": {
        "unit_tests": {
            "status": "completed",
            "output_file": "test-results.txt",
            "json_file": "test-results.json"
        },
        "coverage": {
            "enabled": $COVERAGE,
            "html_report": "coverage/coverage.html",
            "data_file": "coverage/coverage.out"
        },
        "benchmark": {
            "enabled": $BENCHMARK,
            "results_file": "benchmark/benchmark-results.txt"
        },
        "profile": {
            "enabled": $PROFILE,
            "cpu_profile": "profile/cpu.prof",
            "memory_profile": "profile/mem.prof"
        }
    }
}
EOF
    
    log_success "JSON测试报告生成: $json_file"
}

# 生成XML报告
generate_xml_report() {
    log_info "生成XML测试报告..."
    
    local xml_file="$OUTPUT_DIR/test-report-$TIMESTAMP.xml"
    
    cat > "$xml_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<testReport>
    <metadata>
        <generatedAt>$(date -Iseconds)</generatedAt>
        <project>BindDiff</project>
        <version>2.0.0</version>
        <testFramework>go test</testFramework>
    </metadata>
    
    <summary>
        <totalTests>0</totalTests>
        <passedTests>0</passedTests>
        <failedTests>0</failedTests>
        <coveragePercentage>0%</coveragePercentage>
        <executionTimeSeconds>0</executionTimeSeconds>
    </summary>
    
    <results>
        <unitTests status="completed">
            <outputFile>test-results.txt</outputFile>
            <jsonFile>test-results.json</jsonFile>
        </unitTests>
        
        <coverage enabled="$COVERAGE">
            <htmlReport>coverage/coverage.html</htmlReport>
            <dataFile>coverage/coverage.out</dataFile>
        </coverage>
        
        <benchmark enabled="$BENCHMARK">
            <resultsFile>benchmark/benchmark-results.txt</resultsFile>
        </benchmark>
        
        <profile enabled="$PROFILE">
            <cpuProfile>profile/cpu.prof</cpuProfile>
            <memoryProfile>profile/mem.prof</memoryProfile>
        </profile>
    </results>
</testReport>
EOF
    
    log_success "XML测试报告生成: $xml_file"
}

# 清理临时文件
cleanup() {
    if [[ "$NO_CLEANUP" == "true" ]]; then
        log_info "跳过清理临时文件"
        return 0
    fi
    
    log_info "清理临时文件..."
    
    # 清理Go测试缓存
    go clean -testcache >/dev/null 2>&1 || true
    
    # 清理其他临时文件
    find "$OUTPUT_DIR" -name "*.tmp" -delete 2>/dev/null || true
    
    log_info "清理完成"
}

# 主函数
main() {
    log_info "BindDiff 测试报告生成器启动"
    
    # 检查工具
    check_tools
    
    # 运行测试
    local test_exit_code=0
    run_unit_tests || test_exit_code=$?
    
    # 生成覆盖率报告
    generate_coverage
    
    # 运行基准测试
    run_benchmark
    
    # 生成性能分析
    generate_profile
    
    # 生成报告
    case "$FORMAT" in
        "html")
            generate_html_report
            ;;
        "json")
            generate_json_report
            ;;
        "xml")
            generate_xml_report
            ;;
        "all")
            generate_html_report
            generate_json_report
            generate_xml_report
            ;;
    esac
    
    # 清理
    cleanup
    
    # 显示结果摘要
    log_info "测试报告生成完成"
    log_info "输出目录: $OUTPUT_DIR"
    
    if [[ "$FORMAT" == "all" || "$FORMAT" == "html" ]]; then
        local html_report=$(find "$OUTPUT_DIR" -name "test-report-*.html" | head -1)
        if [[ -n "$html_report" ]]; then
            log_success "HTML报告: $html_report"
        fi
    fi
    
    # 返回测试的退出码
    exit $test_exit_code
}

# 运行主函数
main "$@"