package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/models"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
	// 创建上下文，设置超时时间为30秒
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建基础仓库实例
	repo := repository.NewRepository()

	// 定义要批量查询的Gem列表
	gems := []string{
		"rails",
		"rack",
		"activesupport",
		"rake",
		"concurrent-ruby",
		"i18n",
		"minitest",
		"tzinfo",
		"nokogiri",
		"zeitwerk",
	}

	fmt.Println("开始批量获取包信息...")
	startTime := time.Now()

	// 创建批量操作选项，设置最大并发数为5
	options := repository.NewBulkOptions().WithMaxConcurrency(5)

	// 批量获取包信息
	results := repo.BulkGetPackages(ctx, gems, options)

	// 计算耗时
	duration := time.Since(startTime)
	fmt.Printf("批量获取完成，耗时: %v\n\n", duration)

	// 显示结果
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("获取 %s 失败: %v\n", result.Key, result.Error)
			continue
		}

		pkg := result.Value
		fmt.Printf("包名: %s\n", pkg.Name)
		fmt.Printf("  当前版本: %s\n", pkg.Version)
		fmt.Printf("  主页: %s\n", pkg.HomepageURI)
		fmt.Printf("  下载量: %d\n", pkg.Downloads)
		fmt.Printf("  描述: %s\n\n", pkg.Info)
	}

	// 对比顺序执行的时间
	fmt.Println("开始顺序获取包信息进行对比...")
	startTime = time.Now()

	// 顺序获取包信息
	sequentialResults := make([]*repository.BulkResult[*models.PackageInformation], 0, len(gems))
	for _, gemName := range gems {
		pkg, err := repo.GetPackage(ctx, gemName)
		sequentialResults = append(sequentialResults, &repository.BulkResult[*models.PackageInformation]{
			Key:   gemName,
			Value: pkg,
			Error: err,
		})
	}

	// 计算耗时
	sequentialDuration := time.Since(startTime)
	fmt.Printf("顺序获取完成，耗时: %v\n", sequentialDuration)
	fmt.Printf("并发处理比顺序处理快 %.2f 倍\n\n", float64(sequentialDuration)/float64(duration))

	// 演示批量获取版本信息
	fmt.Println("开始批量获取版本信息...")
	startTime = time.Now()

	// 选择前5个gem进行版本查询
	selectedGems := gems[:5]
	versionResults := repo.BulkGetVersions(ctx, selectedGems, options)

	// 计算耗时
	duration = time.Since(startTime)
	fmt.Printf("批量获取版本信息完成，耗时: %v\n\n", duration)

	// 显示版本信息结果
	for _, result := range versionResults {
		if result.Error != nil {
			fmt.Printf("获取 %s 版本信息失败: %v\n", result.Key, result.Error)
			continue
		}

		versions := result.Value
		fmt.Printf("包名: %s\n", result.Key)
		fmt.Printf("  版本数量: %d\n", len(versions))
		if len(versions) > 0 {
			fmt.Printf("  最新几个版本: %s", versions[0].Number)
			for i := 1; i < min(5, len(versions)); i++ {
				fmt.Printf(", %s", versions[i].Number)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// 演示批量获取依赖信息
	fmt.Println("开始批量获取依赖信息...")
	startTime = time.Now()

	// 获取依赖信息
	dependencyResults := repo.BulkGetDependencies(ctx, selectedGems, options)

	// 计算耗时
	duration = time.Since(startTime)
	fmt.Printf("批量获取依赖信息完成，耗时: %v\n\n", duration)

	// 显示依赖信息结果
	for _, result := range dependencyResults {
		if result.Error != nil {
			fmt.Printf("获取 %s 依赖信息失败: %v\n", result.Key, result.Error)
			continue
		}

		dependencies := result.Value
		fmt.Printf("包名: %s\n", result.Key)
		fmt.Printf("  依赖数量: %d\n", len(dependencies))
		if len(dependencies) > 0 {
			fmt.Println("  部分依赖列表:")
			for i := 0; i < min(5, len(dependencies)); i++ {
				dep := dependencies[i]
				fmt.Printf("    - %s (要求: %s)\n", dep.Name, dep.Requirements)
			}
		}
		fmt.Println()
	}
}

// Go 1.20 及以下版本没有内置min函数，手动实现
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
