@echo off
setlocal enabledelayedexpansion

REM BindDiff 测试报告生成脚本 (Windows 版本)
REM 支持生成 HTML、JSON、XML 格式的测试报告

REM 设置UTF-8编码
chcp 65001 >nul 2>&1

REM 显示启动信息
echo BindDiff 测试报告生成器 (Windows)
echo ====================================

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%\.."
set "REPORT_DIR=%PROJECT_ROOT%\test-reports"

REM 获取时间戳
for /f "tokens=1-6 delims=/: " %%a in ('echo %date% %time%') do (
    set "TIMESTAMP=%%c%%a%%b_%%d%%e%%f"
)
set "TIMESTAMP=%TIMESTAMP: =0%"

REM 默认参数
set "FORMAT=html"
set "OUTPUT_DIR=%REPORT_DIR%"
set "VERBOSE=false"
set "COVERAGE=false"
set "BENCHMARK=false"
set "PROFILE=false"
set "NO_CLEANUP=false"

REM 解析命令行参数
:parse_args
if "%1"=="" goto :args_done
if "%1"=="-h" goto :show_help
if "%1"=="--help" goto :show_help
if "%1"=="-f" (
    set "FORMAT=%2"
    shift
    shift
    goto :parse_args
)
if "%1"=="--format" (
    set "FORMAT=%2"
    shift
    shift
    goto :parse_args
)
if "%1"=="-o" (
    set "OUTPUT_DIR=%2"
    shift
    shift
    goto :parse_args
)
if "%1"=="--output" (
    set "OUTPUT_DIR=%2"
    shift
    shift
    goto :parse_args
)
if "%1"=="-v" (
    set "VERBOSE=true"
    shift
    goto :parse_args
)
if "%1"=="--verbose" (
    set "VERBOSE=true"
    shift
    goto :parse_args
)
if "%1"=="-c" (
    set "COVERAGE=true"
    shift
    goto :parse_args
)
if "%1"=="--coverage" (
    set "COVERAGE=true"
    shift
    goto :parse_args
)
if "%1"=="-b" (
    set "BENCHMARK=true"
    shift
    goto :parse_args
)
if "%1"=="--benchmark" (
    set "BENCHMARK=true"
    shift
    goto :parse_args
)
if "%1"=="-p" (
    set "PROFILE=true"
    shift
    goto :parse_args
)
if "%1"=="--profile" (
    set "PROFILE=true"
    shift
    goto :parse_args
)
if "%1"=="--no-cleanup" (
    set "NO_CLEANUP=true"
    shift
    goto :parse_args
)
echo 未知选项: %1
goto :show_help

:args_done

REM 验证格式
if not "%FORMAT%"=="html" if not "%FORMAT%"=="json" if not "%FORMAT%"=="xml" if not "%FORMAT%"=="all" (
    echo 不支持的格式: %FORMAT%
    goto :show_help
)

REM 创建必要目录
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
if not exist "%OUTPUT_DIR%\coverage" mkdir "%OUTPUT_DIR%\coverage"
if not exist "%OUTPUT_DIR%\benchmark" mkdir "%OUTPUT_DIR%\benchmark"
if not exist "%OUTPUT_DIR%\profile" mkdir "%OUTPUT_DIR%\profile"

REM 进入项目根目录
cd /d "%PROJECT_ROOT%"

echo [INFO] 开始生成测试报告...
echo [INFO] 项目根目录: %PROJECT_ROOT%
echo [INFO] 输出目录: %OUTPUT_DIR%
echo [INFO] 报告格式: %FORMAT%

REM 运行单元测试
echo [INFO] 运行单元测试...
set "TEST_OUTPUT=%OUTPUT_DIR%\test-results.txt"
set "JSON_OUTPUT=%OUTPUT_DIR%\test-results.json"

if "%VERBOSE%"=="true" (
    go test -v ./test/... > "%TEST_OUTPUT%" 2>&1
) else (
    go test ./test/... > "%TEST_OUTPUT%" 2>&1
)

set "TEST_EXIT_CODE=%errorlevel%"

REM 生成JSON格式的测试结果
go test -json ./test/... > "%JSON_OUTPUT%" 2>nul

if %TEST_EXIT_CODE% equ 0 (
    echo [SUCCESS] 单元测试通过
) else (
    echo [WARNING] 单元测试有失败项
)

REM 生成覆盖率报告
if "%COVERAGE%"=="true" (
    echo [INFO] 生成覆盖率报告...
    
    set "COVERAGE_OUT=%OUTPUT_DIR%\coverage\coverage.out"
    set "COVERAGE_HTML=%OUTPUT_DIR%\coverage\coverage.html"
    
    go test -race -coverprofile="!COVERAGE_OUT!" -covermode=atomic ./test/...
    
    if exist "!COVERAGE_OUT!" (
        go tool cover -html="!COVERAGE_OUT!" -o "!COVERAGE_HTML!"
        echo [SUCCESS] 覆盖率HTML报告生成: !COVERAGE_HTML!
        
        REM 显示覆盖率摘要
        for /f "tokens=3" %%i in ('go tool cover -func="!COVERAGE_OUT!" ^| findstr "^total:"') do (
            echo [INFO] 总体代码覆盖率: %%i
        )
    ) else (
        echo [WARNING] 覆盖率数据文件未生成
    )
)

REM 运行基准测试
if "%BENCHMARK%"=="true" (
    echo [INFO] 运行基准测试...
    
    set "BENCH_OUTPUT=%OUTPUT_DIR%\benchmark\benchmark-results.txt"
    
    go test -bench=. -benchmem ./test/... > "!BENCH_OUTPUT!" 2>&1
    
    echo [SUCCESS] 基准测试报告生成: !BENCH_OUTPUT!
)

REM 生成性能分析报告
if "%PROFILE%"=="true" (
    echo [INFO] 生成性能分析报告...
    
    set "CPU_PROF=%OUTPUT_DIR%\profile\cpu.prof"
    set "MEM_PROF=%OUTPUT_DIR%\profile\mem.prof"
    
    REM CPU性能分析
    go test -cpuprofile="!CPU_PROF!" -bench=. ./test/core/ >nul 2>&1
    if exist "!CPU_PROF!" (
        echo [SUCCESS] CPU性能分析文件生成: !CPU_PROF!
        echo [INFO] 查看CPU分析: go tool pprof !CPU_PROF!
    )
    
    REM 内存性能分析
    go test -memprofile="!MEM_PROF!" -bench=. ./test/core/ >nul 2>&1
    if exist "!MEM_PROF!" (
        echo [SUCCESS] 内存性能分析文件生成: !MEM_PROF!
        echo [INFO] 查看内存分析: go tool pprof !MEM_PROF!
    )
)

REM 生成HTML报告
if "%FORMAT%"=="html" goto :generate_html
if "%FORMAT%"=="all" goto :generate_html
goto :skip_html

:generate_html
echo [INFO] 生成HTML测试报告...

set "HTML_FILE=%OUTPUT_DIR%\test-report-%TIMESTAMP%.html"

REM 创建简化的HTML报告
(
echo ^<!DOCTYPE html^>
echo ^<html lang="zh-CN"^>
echo ^<head^>
echo     ^<meta charset="UTF-8"^>
echo     ^<meta name="viewport" content="width=device-width, initial-scale=1.0"^>
echo     ^<title^>BindDiff 测试报告^</title^>
echo     ^<style^>
echo         body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
echo         .container { max-width: 1200px; margin: 0 auto; }
echo         .header { background: #667eea; color: white; padding: 2rem; border-radius: 10px; text-align: center; }
echo         .section { background: white; margin: 1rem 0; padding: 1.5rem; border-radius: 8px; box-shadow: 0 2px 5px rgba(0,0,0,0.1^); }
echo         .status-pass { background: #d4edda; color: #155724; padding: 1rem; border-radius: 5px; }
echo         .status-fail { background: #f8d7da; color: #721c24; padding: 1rem; border-radius: 5px; }
echo         pre { background: #f8f9fa; padding: 1rem; border-radius: 4px; overflow-x: auto; }
echo     ^</style^>
echo ^</head^>
echo ^<body^>
echo     ^<div class="container"^>
echo         ^<div class="header"^>
echo             ^<h1^>BindDiff 测试报告^</h1^>
echo             ^<p^>生成时间: %date% %time%^</p^>
echo         ^</div^>
echo         
echo         ^<div class="section"^>
echo             ^<h2^>测试概要^</h2^>
if %TEST_EXIT_CODE% equ 0 (
echo             ^<div class="status-pass"^>单元测试: 通过^</div^>
) else (
echo             ^<div class="status-fail"^>单元测试: 有失败项^</div^>
)
echo         ^</div^>
echo         
echo         ^<div class="section"^>
echo             ^<h2^>测试输出^</h2^>
echo             ^<pre^>
type "%TEST_OUTPUT%"
echo             ^</pre^>
echo         ^</div^>

if "%COVERAGE%"=="true" (
echo         ^<div class="section"^>
echo             ^<h2^>覆盖率报告^</h2^>
echo             ^<p^>详细报告: ^<a href="coverage/coverage.html"^>查看HTML报告^</a^>^</p^>
echo         ^</div^>
)

if "%BENCHMARK%"=="true" (
echo         ^<div class="section"^>
echo             ^<h2^>基准测试^</h2^>
echo             ^<pre^>
type "%OUTPUT_DIR%\benchmark\benchmark-results.txt" 2>nul
echo             ^</pre^>
echo         ^</div^>
)

echo     ^</div^>
echo ^</body^>
echo ^</html^>
) > "%HTML_FILE%"

echo [SUCCESS] HTML测试报告生成: %HTML_FILE%

:skip_html

REM 生成JSON报告
if "%FORMAT%"=="json" goto :generate_json
if "%FORMAT%"=="all" goto :generate_json
goto :skip_json

:generate_json
echo [INFO] 生成JSON测试报告...

set "JSON_FILE=%OUTPUT_DIR%\test-report-%TIMESTAMP%.json"

(
echo {
echo     "report": {
echo         "generated_at": "%date% %time%",
echo         "project": "BindDiff",
echo         "version": "2.0.0",
echo         "test_framework": "go test"
echo     },
echo     "summary": {
echo         "test_status": "%TEST_EXIT_CODE%",
echo         "coverage_enabled": %COVERAGE%,
echo         "benchmark_enabled": %BENCHMARK%,
echo         "profile_enabled": %PROFILE%
echo     },
echo     "files": {
echo         "test_output": "test-results.txt",
echo         "test_json": "test-results.json"
if "%COVERAGE%"=="true" (
echo         ,"coverage_html": "coverage/coverage.html"
echo         ,"coverage_data": "coverage/coverage.out"
)
if "%BENCHMARK%"=="true" (
echo         ,"benchmark_results": "benchmark/benchmark-results.txt"
)
echo     }
echo }
) > "%JSON_FILE%"

echo [SUCCESS] JSON测试报告生成: %JSON_FILE%

:skip_json

REM 生成XML报告
if "%FORMAT%"=="xml" goto :generate_xml
if "%FORMAT%"=="all" goto :generate_xml
goto :skip_xml

:generate_xml
echo [INFO] 生成XML测试报告...

set "XML_FILE=%OUTPUT_DIR%\test-report-%TIMESTAMP%.xml"

(
echo ^<?xml version="1.0" encoding="UTF-8"?^>
echo ^<testReport^>
echo     ^<metadata^>
echo         ^<generatedAt^>%date% %time%^</generatedAt^>
echo         ^<project^>BindDiff^</project^>
echo         ^<version^>2.0.0^</version^>
echo         ^<testFramework^>go test^</testFramework^>
echo     ^</metadata^>
echo     
echo     ^<summary^>
echo         ^<testStatus^>%TEST_EXIT_CODE%^</testStatus^>
echo         ^<coverageEnabled^>%COVERAGE%^</coverageEnabled^>
echo         ^<benchmarkEnabled^>%BENCHMARK%^</benchmarkEnabled^>
echo         ^<profileEnabled^>%PROFILE%^</profileEnabled^>
echo     ^</summary^>
echo     
echo     ^<files^>
echo         ^<testOutput^>test-results.txt^</testOutput^>
echo         ^<testJson^>test-results.json^</testJson^>
if "%COVERAGE%"=="true" (
echo         ^<coverageHtml^>coverage/coverage.html^</coverageHtml^>
echo         ^<coverageData^>coverage/coverage.out^</coverageData^>
)
if "%BENCHMARK%"=="true" (
echo         ^<benchmarkResults^>benchmark/benchmark-results.txt^</benchmarkResults^>
)
echo     ^</files^>
echo ^</testReport^>
) > "%XML_FILE%"

echo [SUCCESS] XML测试报告生成: %XML_FILE%

:skip_xml

REM 清理临时文件
if not "%NO_CLEANUP%"=="true" (
    echo [INFO] 清理临时文件...
    go clean -testcache >nul 2>&1
)

REM 显示结果摘要
echo [INFO] 测试报告生成完成
echo [INFO] 输出目录: %OUTPUT_DIR%

REM 返回测试的退出码
exit /b %TEST_EXIT_CODE%

:show_help
echo BindDiff 测试报告生成器 (Windows 版本)
echo.
echo 用法: %0 [选项]
echo.
echo 选项:
echo     -h, --help          显示此帮助信息
echo     -f, --format        报告格式 (html^|json^|xml^|all) [默认: html]
echo     -o, --output        输出目录 [默认: test-reports]
echo     -v, --verbose       详细输出
echo     -c, --coverage      生成覆盖率报告
echo     -b, --benchmark     运行基准测试
echo     -p, --profile       生成性能分析报告
echo     --no-cleanup        不清理临时文件
echo.
echo 示例:
echo     %0                           # 生成基本HTML报告
echo     %0 -f all -c -b             # 生成所有格式报告，包含覆盖率和基准测试
echo     %0 -f json -o custom-dir    # 生成JSON格式报告到自定义目录
exit /b 0