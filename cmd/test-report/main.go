package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ReportConfig 测试报告配置
type ReportConfig struct {
	Report struct {
		ProjectName string `yaml:"project_name"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	} `yaml:"report"`
	Output struct {
		Directory       string   `yaml:"directory"`
		Formats         []string `yaml:"formats"`
		TimestampFormat string   `yaml:"timestamp_format"`
	} `yaml:"output"`
	Testing struct {
		UnitTests struct {
			Enabled  bool     `yaml:"enabled"`
			Verbose  bool     `yaml:"verbose"`
			Timeout  string   `yaml:"timeout"`
			Packages []string `yaml:"packages"`
		} `yaml:"unit_tests"`
		Coverage struct {
			Enabled         bool     `yaml:"enabled"`
			Mode            string   `yaml:"mode"`
			Threshold       float64  `yaml:"threshold"`
			ExcludePatterns []string `yaml:"exclude_patterns"`
		} `yaml:"coverage"`
		Benchmark struct {
			Enabled       bool   `yaml:"enabled"`
			Count         int    `yaml:"count"`
			Timeout       string `yaml:"timeout"`
			MemoryProfile bool   `yaml:"memory_profile"`
			CPUProfile    bool   `yaml:"cpu_profile"`
		} `yaml:"benchmark"`
		Profiling struct {
			Enabled       bool `yaml:"enabled"`
			CPUProfile    bool `yaml:"cpu_profile"`
			MemoryProfile bool `yaml:"memory_profile"`
			TraceProfile  bool `yaml:"trace_profile"`
		} `yaml:"profiling"`
	} `yaml:"testing"`
	Content struct {
		HTML struct {
			Title              string `yaml:"title"`
			Theme              string `yaml:"theme"`
			IncludeCharts      bool   `yaml:"include_charts"`
			IncludeSourceLinks bool   `yaml:"include_source_links"`
		} `yaml:"html"`
		Sections struct {
			Summary     bool `yaml:"summary"`
			TestResults bool `yaml:"test_results"`
			Coverage    bool `yaml:"coverage"`
			Benchmarks  bool `yaml:"benchmarks"`
			Performance bool `yaml:"performance"`
		} `yaml:"sections"`
		Metrics struct {
			TestCount     bool `yaml:"test_count"`
			PassRate      bool `yaml:"pass_rate"`
			CoverageRate  bool `yaml:"coverage_rate"`
			ExecutionTime bool `yaml:"execution_time"`
			MemoryUsage   bool `yaml:"memory_usage"`
		} `yaml:"metrics"`
	} `yaml:"content"`
	Advanced struct {
		Parallel struct {
			Enabled bool `yaml:"enabled"`
			Workers int  `yaml:"workers"`
		} `yaml:"parallel"`
		Cache struct {
			Enabled   bool `yaml:"enabled"`
			TestCache bool `yaml:"test_cache"`
		} `yaml:"cache"`
		Debug struct {
			Enabled        bool `yaml:"enabled"`
			VerboseLogging bool `yaml:"verbose_logging"`
			SaveRawOutput  bool `yaml:"save_raw_output"`
		} `yaml:"debug"`
	} `yaml:"advanced"`
}

// TestResult 测试结果
type TestResult struct {
	Package    string
	Test       string
	Status     string
	Duration   time.Duration
	Output     string
	Benchmark  bool
	MemoryUsed int64
}

// TestSummary 测试摘要
type TestSummary struct {
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Duration     time.Duration
	Coverage     float64
}

// ReportData 报告数据
type ReportData struct {
	Config           *ReportConfig
	Summary          TestSummary
	Results          []TestResult
	CoverageDetails  string
	BenchmarkResults string
	GeneratedAt      time.Time
	OutputDir        string
}

var (
	configFile = flag.String("config", "configs/test-report.yaml", "配置文件路径")
	format     = flag.String("format", "", "报告格式 (html,json,xml,all)")
	output     = flag.String("output", "", "输出目录")
	coverage   = flag.Bool("coverage", false, "生成覆盖率报告")
	benchmark  = flag.Bool("benchmark", false, "运行基准测试")
	profile    = flag.Bool("profile", false, "生成性能分析")
)

// initializeConfig 初始化配置
// 加载配置文件并应用命令行参数覆盖
func initializeConfig() (*ReportConfig, error) {
	// 加载基础配置
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Printf("警告: 无法加载配置文件 %s: %v", *configFile, err)
		config = defaultConfig()
	}

	// 应用命令行参数覆盖
	applyCommandLineOverrides(config)

	// 验证配置的有效性
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return config, nil
}

// applyCommandLineOverrides 应用命令行参数覆盖
func applyCommandLineOverrides(config *ReportConfig) {
	if *format != "" {
		config.Output.Formats = strings.Split(*format, ",")
	}
	if *output != "" {
		config.Output.Directory = *output
	}
	if *coverage {
		config.Testing.Coverage.Enabled = true
	}
	if *benchmark {
		config.Testing.Benchmark.Enabled = true
	}
	if *profile {
		config.Testing.Profiling.Enabled = true
	}
}

// validateConfig 验证配置的有效性
func validateConfig(config *ReportConfig) error {
	if config.Output.Directory == "" {
		return fmt.Errorf("输出目录不能为空")
	}
	if len(config.Output.Formats) == 0 {
		config.Output.Formats = []string{"html"} // 默认格式
	}

	// 验证支持的报告格式
	supportedFormats := map[string]bool{"html": true, "json": true, "xml": true}
	for _, format := range config.Output.Formats {
		if !supportedFormats[format] {
			return fmt.Errorf("不支持的报告格式: %s", format)
		}
	}

	return nil
}

// printStartupInfo 显示启动信息
func printStartupInfo(config *ReportConfig) {
	fmt.Printf("🚀 BindDiff 测试报告生成器启动\n")
	fmt.Printf("📋 项目: %s v%s\n", config.Report.ProjectName, config.Report.Version)
	fmt.Printf("📁 输出目录: %s\n", config.Output.Directory)
	fmt.Printf("📄 报告格式: %v\n", config.Output.Formats)
}

// setupOutputDirectories 设置输出目录结构
func setupOutputDirectories(config *ReportConfig) error {
	// 创建主输出目录
	if err := os.MkdirAll(config.Output.Directory, 0755); err != nil {
		return fmt.Errorf("创建主输出目录失败: %v", err)
	}

	// 创建子目录
	subdirs := []string{"coverage", "benchmark", "profile"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(config.Output.Directory, subdir)
		if err := os.MkdirAll(subdirPath, 0755); err != nil {
			log.Printf("警告: 创建子目录 %s 失败: %v", subdir, err)
		}
	}

	return nil
}

// executeTestsAndCollectData 执行测试并收集数据
// 这是主要的测试执行流程
func executeTestsAndCollectData(config *ReportConfig) (*ReportData, error) {
	reportData := &ReportData{
		Config:      config,
		GeneratedAt: time.Now(),
		OutputDir:   config.Output.Directory,
	}

	// 按顺序执行各种测试
	testSteps := []struct {
		name    string
		enabled bool
		execute func(*ReportData) error
	}{
		{"单元测试", config.Testing.UnitTests.Enabled, runUnitTests},
		{"覆盖率测试", config.Testing.Coverage.Enabled, runCoverageTests},
		{"基准测试", config.Testing.Benchmark.Enabled, runBenchmarkTests},
		{"性能分析", config.Testing.Profiling.Enabled, runProfilingTests},
	}

	for _, step := range testSteps {
		if step.enabled {
			fmt.Printf("🧪 正在执行%s...\n", step.name)
			if err := step.execute(reportData); err != nil {
				log.Printf("警告: %s失败: %v", step.name, err)
			}
		}
	}

	return reportData, nil
}

// generateAllReports 生成所有格式的报告
func generateAllReports(reportData *ReportData) error {
	var errors []string

	// 报告生成器映射
	reportGenerators := map[string]func(*ReportData) error{
		"html": generateHTMLReport,
		"json": generateJSONReport,
		"xml":  generateXMLReport,
	}

	// 生成各种格式的报告
	for _, format := range reportData.Config.Output.Formats {
		if generator, exists := reportGenerators[format]; exists {
			if err := generator(reportData); err != nil {
				errorMsg := fmt.Sprintf("生成%s报告失败: %v", strings.ToUpper(format), err)
				errors = append(errors, errorMsg)
				log.Print(errorMsg)
			} else {
				fmt.Printf("✅ %s报告已生成\n", strings.ToUpper(format))
			}
		} else {
			errorMsg := fmt.Sprintf("不支持的报告格式: %s", format)
			errors = append(errors, errorMsg)
			log.Print(errorMsg)
		}
	}

	// 如果有错误，返回聚合错误信息
	if len(errors) > 0 {
		return fmt.Errorf("报告生成错误: %s", strings.Join(errors, "; "))
	}

	return nil
}

// printFinalSummary 显示最终摘要
func printFinalSummary(reportData *ReportData) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 测试报告摘要")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("📊 总测试数: %d\n", reportData.Summary.TotalTests)
	fmt.Printf("✅ 通过数: %d\n", reportData.Summary.PassedTests)
	if reportData.Summary.FailedTests > 0 {
		fmt.Printf("❌ 失败数: %d\n", reportData.Summary.FailedTests)
	}
	if reportData.Summary.SkippedTests > 0 {
		fmt.Printf("⏭️  跳过数: %d\n", reportData.Summary.SkippedTests)
	}
	if reportData.Summary.Coverage > 0 {
		fmt.Printf("📈 覆盖率: %.1f%%\n", reportData.Summary.Coverage)
	}
	fmt.Printf("📁 输出目录: %s\n", reportData.OutputDir)
	fmt.Println(strings.Repeat("=", 60))

	if reportData.Summary.FailedTests > 0 {
		fmt.Println("⚠️  存在失败的测试用例，请检查详细报告")
	} else {
		fmt.Println("🎉 所有测试均通过！")
	}
}

// setExitCode 根据测试结果设置退出码
func setExitCode(reportData *ReportData) {
	if reportData.Summary.FailedTests > 0 {
		os.Exit(1)
	}
	// 成功退出，退出码为 0
}

// main 程序入口点
// 解析命令行参数，加载配置，运行测试并生成报告
func main() {
	flag.Parse()

	// 初始化并验证配置
	config, err := initializeConfig()
	if err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}

	// 显示启动信息
	printStartupInfo(config)

	// 设置输出环境
	if err := setupOutputDirectories(config); err != nil {
		log.Fatalf("设置输出目录失败: %v", err)
	}

	// 执行测试并收集数据
	reportData, err := executeTestsAndCollectData(config)
	if err != nil {
		log.Fatalf("测试执行失败: %v", err)
	}

	// 生成所有格式的报告
	if err := generateAllReports(reportData); err != nil {
		log.Printf("报告生成过程中出现错误: %v", err)
	}

	// 显示最终摘要
	printFinalSummary(reportData)

	// 根据测试结果设置退出码
	setExitCode(reportData)
}

// loadConfig 加载配置文件
// 从指定的YAML文件中加载测试报告配置
func loadConfig(filename string) (*ReportConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config ReportConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// defaultConfig 生成默认配置
// 当配置文件不存在或加载失败时使用
func defaultConfig() *ReportConfig {
	config := &ReportConfig{}

	// 设置项目基本信息
	config.Report.ProjectName = "BindDiff"
	config.Report.Version = "2.0.0"
	config.Report.Description = "高性能二进制差异分析工具"

	// 设置输出配置
	config.Output.Directory = "test-reports"
	config.Output.Formats = []string{"html"}
	config.Output.TimestampFormat = "20060102_150405"

	// 设置测试配置默认值
	config.Testing.UnitTests.Enabled = true
	config.Testing.UnitTests.Verbose = false
	config.Testing.UnitTests.Timeout = "10m"
	config.Testing.UnitTests.Packages = []string{"./test/..."}

	// 设置覆盖率配置
	config.Testing.Coverage.Enabled = false
	config.Testing.Coverage.Mode = "atomic"
	config.Testing.Coverage.Threshold = 70.0

	// 设置基准测试配置
	config.Testing.Benchmark.Enabled = false
	config.Testing.Benchmark.Count = 1
	config.Testing.Benchmark.Timeout = "30m"

	// 设置HTML报告配置
	config.Content.HTML.Title = "BindDiff 测试报告"
	config.Content.HTML.Theme = "modern"

	return config
}

// runUnitTests 运行单元测试
// 执行所有指定包的单元测试并收集结果
func runUnitTests(reportData *ReportData) error {
	// 获取测试包列表
	packages := getTestPackages(reportData.Config)

	// 构建测试命令参数
	args := buildTestArgs(reportData.Config, packages)

	// 执行测试命令
	output, err := executeTestCommand(args)
	if err != nil {
		log.Printf("测试执行警告: %v", err)
	}

	// 解析测试输出结果
	reportData.Results = parseTestOutput(string(output))

	// 统计测试结果
	calculateTestSummary(reportData)

	return nil
}

// getTestPackages 获取测试包列表
func getTestPackages(config *ReportConfig) []string {
	packages := config.Testing.UnitTests.Packages
	if len(packages) == 0 {
		// 默认测试所有test目录下的包
		packages = []string{"./test/..."}
	}
	return packages
}

// buildTestArgs 构建测试命令参数
func buildTestArgs(config *ReportConfig, packages []string) []string {
	args := []string{"test"}

	// 添加详细输出参数
	if config.Testing.UnitTests.Verbose {
		args = append(args, "-v")
	}

	// 添加超时参数
	if config.Testing.UnitTests.Timeout != "" {
		args = append(args, "-timeout", config.Testing.UnitTests.Timeout)
	}

	// 添加测试包
	args = append(args, packages...)

	return args
}

// executeTestCommand 执行测试命令
func executeTestCommand(args []string) ([]byte, error) {
	cmd := exec.Command("go", args...)
	return cmd.CombinedOutput()
}

// calculateTestSummary 统计测试结果
func calculateTestSummary(reportData *ReportData) {
	for _, result := range reportData.Results {
		reportData.Summary.TotalTests++
		switch result.Status {
		case "PASS":
			reportData.Summary.PassedTests++
		case "FAIL":
			reportData.Summary.FailedTests++
		case "SKIP":
			reportData.Summary.SkippedTests++
		}
		reportData.Summary.Duration += result.Duration
	}
}

// runCoverageTests 运行覆盖率测试
// 生成代码覆盖率报告并分析覆盖率数据
func runCoverageTests(reportData *ReportData) error {
	coverageFile := filepath.Join(reportData.OutputDir, "coverage", "coverage.out")
	htmlFile := filepath.Join(reportData.OutputDir, "coverage", "coverage.html")

	// 执行覆盖率测试
	if err := executeCoverageTest(coverageFile); err != nil {
		return fmt.Errorf("执行覆盖率测试失败: %v", err)
	}

	// 生成HTML覆盖率报告
	if err := generateCoverageHTML(coverageFile, htmlFile); err != nil {
		return fmt.Errorf("生成HTML覆盖率报告失败: %v", err)
	}

	// 解析覆盖率数据
	if err := parseCoverageData(coverageFile, reportData); err != nil {
		return fmt.Errorf("解析覆盖率数据失败: %v", err)
	}

	return nil
}

// executeCoverageTest 执行覆盖率测试
func executeCoverageTest(coverageFile string) error {
	args := []string{
		"test",
		"-race",
		"-coverprofile=" + coverageFile,
		"-covermode=atomic",
		"./test/...",
	}

	cmd := exec.Command("go", args...)
	_, err := cmd.CombinedOutput()
	return err
}

// generateCoverageHTML 生成HTML覆盖率报告
func generateCoverageHTML(coverageFile, htmlFile string) error {
	cmd := exec.Command("go", "tool", "cover",
		"-html="+coverageFile,
		"-o", htmlFile)
	_, err := cmd.CombinedOutput()
	return err
}

// parseCoverageData 解析覆盖率数据
func parseCoverageData(coverageFile string, reportData *ReportData) error {
	// 获取覆盖率百分比
	cmd := exec.Command("go", "tool", "cover", "-func="+coverageFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	reportData.CoverageDetails = string(output)

	// 解析总体覆盖率
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "total:") {
			if coverage := extractCoveragePercentage(line); coverage >= 0 {
				reportData.Summary.Coverage = coverage
				break
			}
		}
	}

	return nil
}

// extractCoveragePercentage 从覆盖率行中提取百分比
func extractCoveragePercentage(line string) float64 {
	parts := strings.Fields(line)
	if len(parts) >= 3 {
		coverageStr := strings.TrimSuffix(parts[2], "%")
		if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
			return coverage
		}
	}
	return -1 // 表示解析失败
}

// runBenchmarkTests 运行基准测试
func runBenchmarkTests(reportData *ReportData) error {
	benchmarkFile := filepath.Join(reportData.OutputDir, "benchmark", "benchmark-results.txt")

	args := []string{"test", "-bench=.", "-benchmem"}
	if reportData.Config.Testing.Benchmark.Count > 0 {
		args = append(args, fmt.Sprintf("-count=%d", reportData.Config.Testing.Benchmark.Count))
	}
	if reportData.Config.Testing.Benchmark.Timeout != "" {
		args = append(args, "-timeout", reportData.Config.Testing.Benchmark.Timeout)
	}
	args = append(args, "./test/...")

	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	reportData.BenchmarkResults = string(output)

	// 使用UTF-8编码保存基准测试结果到文件
	file, err := os.Create(benchmarkFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	_, writeErr := writer.Write(output)
	return writeErr
}

// runProfilingTests 运行性能分析
func runProfilingTests(reportData *ReportData) error {
	profileDir := filepath.Join(reportData.OutputDir, "profile")

	if reportData.Config.Testing.Profiling.CPUProfile {
		cpuProfileFile := filepath.Join(profileDir, "cpu.prof")
		args := []string{"test", "-cpuprofile=" + cpuProfileFile, "-bench=.", "./test/core/"}
		cmd := exec.Command("go", args...)
		if _, err := cmd.CombinedOutput(); err != nil {
			log.Printf("CPU性能分析失败: %v", err)
		}
	}

	if reportData.Config.Testing.Profiling.MemoryProfile {
		memProfileFile := filepath.Join(profileDir, "mem.prof")
		args := []string{"test", "-memprofile=" + memProfileFile, "-bench=.", "./test/core/"}
		cmd := exec.Command("go", args...)
		if _, err := cmd.CombinedOutput(); err != nil {
			log.Printf("内存性能分析失败: %v", err)
		}
	}

	return nil
}

// parseTestOutput 解析测试输出
func parseTestOutput(output string) []TestResult {
	var results []TestResult
	lines := strings.Split(output, "\n")

	// 简单的测试结果解析
	passRegex := regexp.MustCompile(`^PASS\s+(.+)\s+(\d+\.\d+s)`)
	failRegex := regexp.MustCompile(`^FAIL\s+(.+)\s+(\d+\.\d+s)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := passRegex.FindStringSubmatch(line); matches != nil {
			duration, _ := time.ParseDuration(matches[2])
			results = append(results, TestResult{
				Package:  matches[1],
				Status:   "PASS",
				Duration: duration,
			})
		} else if matches := failRegex.FindStringSubmatch(line); matches != nil {
			duration, _ := time.ParseDuration(matches[2])
			results = append(results, TestResult{
				Package:  matches[1],
				Status:   "FAIL",
				Duration: duration,
			})
		}
	}

	return results
}

// generateHTMLReport 生成HTML报告
func generateHTMLReport(data *ReportData) error {
	timestamp := data.GeneratedAt.Format("20060102_150405")
	filename := filepath.Join(data.OutputDir, fmt.Sprintf("test-report-%s.html", timestamp))

	tmpl := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Config.Content.HTML.Title}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', system-ui, sans-serif; line-height: 1.6; color: #333; background: #f5f7fa; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 2rem; border-radius: 12px; margin-bottom: 2rem; text-align: center; box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .header h1 { font-size: 2.5rem; margin-bottom: 0.5rem; font-weight: 600; }
        .header p { font-size: 1.1rem; opacity: 0.9; }
        .section { background: white; margin-bottom: 2rem; padding: 2rem; border-radius: 12px; box-shadow: 0 2px 15px rgba(0,0,0,0.08); border: 1px solid #e1e8ed; }
        .section h2 { color: #2c3e50; margin-bottom: 1.5rem; padding-bottom: 0.75rem; border-bottom: 3px solid #3498db; font-size: 1.5rem; font-weight: 600; }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1.5rem; margin: 1.5rem 0; }
        .stat-card { background: #f8f9fa; padding: 1.5rem; border-radius: 8px; text-align: center; border-left: 4px solid #3498db; }
        .stat-card.success { border-left-color: #2ecc71; background: #d5f4e6; }
        .stat-card.warning { border-left-color: #f39c12; background: #fef9e7; }
        .stat-card.danger { border-left-color: #e74c3c; background: #fadbd8; }
        .stat-number { font-size: 2rem; font-weight: bold; color: #2c3e50; }
        .stat-label { font-size: 0.9rem; color: #7f8c8d; margin-top: 0.5rem; text-transform: uppercase; letter-spacing: 0.5px; }
        .coverage-bar { background: #ecf0f1; height: 20px; border-radius: 10px; overflow: hidden; margin: 1rem 0; }
        .coverage-fill { height: 100%; background: linear-gradient(90deg, #2ecc71, #27ae60); transition: width 0.3s ease; }
        .test-results { margin-top: 1.5rem; }
        .test-item { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem; margin: 0.5rem 0; border-radius: 6px; border: 1px solid #ecf0f1; }
        .test-item.pass { background: #d5f4e6; border-color: #2ecc71; }
        .test-item.fail { background: #fadbd8; border-color: #e74c3c; }
        .test-name { font-weight: 500; }
        .test-duration { color: #7f8c8d; font-size: 0.9rem; }
        pre { background: #2c3e50; color: #ecf0f1; padding: 1.5rem; border-radius: 8px; overflow-x: auto; font-size: 0.9rem; line-height: 1.4; }
        .footer { text-align: center; padding: 2rem; color: #7f8c8d; }
        .badge { display: inline-block; padding: 0.25rem 0.75rem; border-radius: 12px; font-size: 0.8rem; font-weight: 600; text-transform: uppercase; }
        .badge.success { background: #2ecc71; color: white; }
        .badge.danger { background: #e74c3c; color: white; }
        @media (max-width: 768px) {
            .container { padding: 10px; }
            .header { padding: 1.5rem; }
            .header h1 { font-size: 2rem; }
            .stats-grid { grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 1rem; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Config.Content.HTML.Title}}</h1>
            <p>{{.Config.Report.Description}}</p>
            <p>生成时间: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        </div>
        
        <div class="section">
            <h2>📊 测试概览</h2>
            <div class="stats-grid">
                <div class="stat-card {{if eq .Summary.FailedTests 0}}success{{else}}danger{{end}}">
                    <div class="stat-number">{{.Summary.TotalTests}}</div>
                    <div class="stat-label">测试总数</div>
                </div>
                <div class="stat-card success">
                    <div class="stat-number">{{.Summary.PassedTests}}</div>
                    <div class="stat-label">通过测试</div>
                </div>
                {{if gt .Summary.FailedTests 0}}
                <div class="stat-card danger">
                    <div class="stat-number">{{.Summary.FailedTests}}</div>
                    <div class="stat-label">失败测试</div>
                </div>
                {{end}}
                {{if gt .Summary.Coverage 0}}
                <div class="stat-card {{if ge .Summary.Coverage 70}}success{{else if ge .Summary.Coverage 50}}warning{{else}}danger{{end}}">
                    <div class="stat-number">{{printf "%.1f%%" .Summary.Coverage}}</div>
                    <div class="stat-label">代码覆盖率</div>
                </div>
                {{end}}
            </div>
            
            {{if gt .Summary.Coverage 0}}
            <h3>覆盖率详情</h3>
            <div class="coverage-bar">
                <div class="coverage-fill" style="width: {{.Summary.Coverage}}%"></div>
            </div>
            <p>当前覆盖率: {{printf "%.1f%%" .Summary.Coverage}} 
            {{if ge .Summary.Coverage 70}}
                <span class="badge success">优秀</span>
            {{else if ge .Summary.Coverage 50}}
                <span class="badge warning">良好</span>
            {{else}}
                <span class="badge danger">需要改进</span>
            {{end}}
            </p>
            {{end}}
        </div>
        
        {{if .Results}}
        <div class="section">
            <h2>📝 测试结果</h2>
            <div class="test-results">
                {{range .Results}}
                <div class="test-item {{if eq .Status "PASS"}}pass{{else}}fail{{end}}">
                    <div class="test-name">{{.Package}}</div>
                    <div class="test-duration">{{.Duration}}</div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
        
        {{if .CoverageDetails}}
        <div class="section">
            <h2>📊 详细覆盖率</h2>
            <pre>{{.CoverageDetails}}</pre>
        </div>
        {{end}}
        
        {{if .BenchmarkResults}}
        <div class="section">
            <h2>⚡ 基准测试结果</h2>
            <pre>{{.BenchmarkResults}}</pre>
        </div>
        {{end}}
        
        <div class="footer">
            <p>{{.Config.Report.ProjectName}} v{{.Config.Report.Version}}</p>
            <p>测试报告生成器 - 让测试结果一目了然</p>
        </div>
    </div>
</body>
</html>`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 确保使用UTF-8编码
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	err = t.Execute(writer, data)
	return err
}

// generateJSONReport 生成JSON报告
func generateJSONReport(data *ReportData) error {
	timestamp := data.GeneratedAt.Format("20060102_150405")
	filename := filepath.Join(data.OutputDir, fmt.Sprintf("test-report-%s.json", timestamp))

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// 使用UTF-8编码写入JSON文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	_, err = writer.Write(jsonData)
	return err
}

// generateXMLReport 生成XML报告
func generateXMLReport(data *ReportData) error {
	timestamp := data.GeneratedAt.Format("20060102_150405")
	filename := filepath.Join(data.OutputDir, fmt.Sprintf("test-report-%s.xml", timestamp))

	xmlData, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	header := []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	xmlData = append(header, xmlData...)

	// 使用UTF-8编码写入XML文件
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	_, err = writer.Write(xmlData)
	return err
}
