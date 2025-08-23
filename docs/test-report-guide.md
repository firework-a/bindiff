# BindDiff 测试报告生成示例

本文档展示如何生成 BindDiff 项目的测试报告。

## 快速开始

### 1. 使用 Makefile（推荐）

```bash
# 生成基本 HTML 测试报告
make test-report-quick

# 生成完整的 HTML 测试报告（包含覆盖率）
make test-report

# 生成综合测试报告（所有格式 + 覆盖率 + 基准测试 + 性能分析）
make test-report-all
```

### 2. Windows 批处理脚本

```cmd
# 生成 HTML 报告
scripts\test-report.bat -f html

# 生成带覆盖率的报告
scripts\test-report.bat -f html -c

# 生成完整报告
scripts\test-report.bat -f all -c -b -p
```

### 3. Linux/macOS Shell 脚本

```bash
# 生成 HTML 报告
scripts/test-report.sh -f html

# 生成带覆盖率的报告
scripts/test-report.sh -f html -c

# 生成完整报告
scripts/test-report.sh -f all -c -b -p
```

### 4. Go 报告生成器（高级功能）

```bash
# 编译报告生成器
go build -o test-reporter ./cmd/test-report/

# 使用默认配置
./test-reporter

# 指定配置文件
./test-reporter -config configs/test-report.yaml

# 命令行参数覆盖
./test-reporter -format html,json -coverage -benchmark -profile
```

## 报告类型

### HTML 报告
- 美观的交互式界面
- 实时图表和统计信息
- 测试结果可视化
- 覆盖率进度条
- 响应式设计，支持移动端

### JSON 报告
- 结构化数据格式
- 便于 CI/CD 集成
- 自动化处理友好
- 包含完整的测试元数据

### XML 报告
- 标准 XML 格式
- 兼容各种测试工具
- 企业级集成支持

## 报告内容

生成的测试报告包含以下信息：

### 📊 测试概览
- 总测试数量
- 通过/失败/跳过统计
- 整体执行时间
- 测试成功率

### 📈 代码覆盖率
- 总体覆盖率百分比
- 各模块覆盖率详情
- 覆盖率可视化图表
- 未覆盖代码定位

### ⚡ 性能基准
- 基准测试结果
- 内存使用分析
- 性能对比数据
- 执行时间趋势

### 🔍 性能分析
- CPU 使用分析
- 内存分配分析
- 性能瓶颈识别
- 优化建议

## 配置选项

测试报告生成器支持丰富的配置选项，通过 `configs/test-report.yaml` 文件进行配置：

```yaml
# 报告基本信息
report:
  project_name: "BindDiff"
  version: "2.0.0"
  description: "高性能二进制差异分析工具"

# 输出设置  
output:
  directory: "test-reports"
  formats: ["html", "json"]

# 测试配置
testing:
  unit_tests:
    enabled: true
    verbose: true
  coverage:
    enabled: true
    threshold: 70
  benchmark:
    enabled: true
    count: 3
```

## 输出示例

### 目录结构

```
test-reports/
├── test-report-20240115_143052.html    # HTML报告
├── test-report-20240115_143052.json    # JSON报告
├── test-report-20240115_143052.xml     # XML报告
├── coverage/
│   ├── coverage.html                   # 覆盖率HTML报告
│   └── coverage.out                    # 覆盖率数据文件
├── benchmark/
│   └── benchmark-results.txt           # 基准测试结果
└── profile/
    ├── cpu.prof                        # CPU性能分析
    └── mem.prof                        # 内存性能分析
```

### HTML 报告截图说明

HTML 报告包含以下部分：

1. **页面头部**：项目信息和生成时间
2. **测试概览**：关键指标卡片展示
3. **覆盖率仪表板**：可视化覆盖率进度
4. **测试结果列表**：每个测试的详细状态
5. **基准测试图表**：性能数据可视化
6. **详细日志**：完整的测试输出

## CI/CD 集成

### GitHub Actions 示例

```yaml
name: Test Report
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: 1.21
    
    - name: Run tests and generate report
      run: make test-report-all
    
    - name: Upload test reports
      uses: actions/upload-artifact@v3
      with:
        name: test-reports
        path: test-reports/
```

### Jenkins Pipeline 示例

```groovy
pipeline {
    agent any
    stages {
        stage('Test') {
            steps {
                sh 'make test-report-all'
                publishHTML([
                    allowMissing: false,
                    alwaysLinkToLastBuild: true,
                    keepAll: true,
                    reportDir: 'test-reports',
                    reportFiles: 'test-report-*.html',
                    reportName: 'Test Report'
                ])
            }
        }
    }
}
```

## 最佳实践

### 1. 定期生成报告
- 在每次代码提交后生成测试报告
- 监控覆盖率变化趋势
- 跟踪性能回归

### 2. 设置覆盖率阈值
- 配置最低覆盖率要求（推荐 70%+）
- 对重要模块设置更高标准
- 在 CI/CD 中强制覆盖率检查

### 3. 性能基线管理
- 建立性能基线数据
- 监控性能变化趋势
- 及时发现性能回归

### 4. 报告存档
- 保留历史测试报告
- 对比不同版本的测试结果
- 分析质量改进趋势

## 故障排除

### 常见问题

1. **脚本权限错误**
   ```bash
   chmod +x scripts/test-report.sh
   ```

2. **Go 工具链缺失**
   ```bash
   go version  # 确认 Go 版本 >= 1.21
   ```

3. **依赖缺失**
   ```bash
   go mod download
   go mod tidy
   ```

4. **磁盘空间不足**
   - 清理旧的测试报告
   - 检查可用磁盘空间

### 调试选项

```bash
# 启用详细日志
./test-reporter -config configs/test-report.yaml -verbose

# 保留原始输出
scripts/test-report.sh --no-cleanup -v
```

## 技术支持

如遇到问题，请：

1. 查看详细的错误日志
2. 确认系统环境和依赖
3. 参考项目文档和示例
4. 提交 Issue 或联系开发团队

---

*本文档随项目版本更新，请以最新版本为准。*