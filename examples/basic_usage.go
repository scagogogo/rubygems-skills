package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
	// 创建一个5秒超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 基本用法
	basicUsage(ctx)

	// 使用镜像源
	useMirror(ctx)

	// 使用重试策略
	useRetry(ctx)

	// 使用Token认证
	useToken(ctx)

	// 搜索包
	searchPackage(ctx)

	// 获取包的版本列表
	getVersions(ctx)

	// 获取下载统计
	getDownloadStats(ctx)
}

func basicUsage(ctx context.Context) {
	fmt.Println("=== 基本用法 ===")
	repo := repository.NewRepository()

	pkg, err := repo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}

	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Printf("作者: %s\n", pkg.Authors)
	fmt.Printf("简介: %s\n", pkg.Info)
	fmt.Printf("下载量: %d\n", pkg.Downloads)
	fmt.Printf("主页: %s\n", pkg.HomepageURI)
	fmt.Println()
}

func useMirror(ctx context.Context) {
	fmt.Println("=== 使用镜像源 ===")

	// 使用Ruby中国镜像源
	repo := repository.NewRubyChinaRepository()

	pkg, err := repo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}

	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Println()

	// 也可以尝试其他镜像源
	// repo = repository.NewTSingHuaRepository()
	// repo = repository.NewAliYunRepository()
}

func useRetry(ctx context.Context) {
	fmt.Println("=== 使用重试策略 ===")

	// 自定义重试策略
	retryOptions := repository.NewDefaultRetryOptions().
		WithMaxAttempts(3).
		WithWaitTime(1 * time.Second)

	options := repository.NewOptions().SetRetryOptions(retryOptions)
	repo := repository.NewRepository(options)

	pkg, err := repo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}

	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Println()
}

func useToken(ctx context.Context) {
	fmt.Println("=== 使用Token认证 ===")

	// 请替换为您的实际Token
	options := repository.NewOptions().SetToken("your-api-token")
	repo := repository.NewRepository(options)

	// 这里仅为示例，未使用实际Token
	fmt.Println("注意: 请替换为您的实际Token")
	fmt.Println("使用Token认证的Repository对象已创建: ", repo != nil)
	fmt.Println()
}

func searchPackage(ctx context.Context) {
	fmt.Println("=== 搜索包 ===")
	repo := repository.NewRepository()

	// 搜索包含"rails"的包，第一页
	packages, err := repo.Search(ctx, "rails", 1)
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
		return
	}

	fmt.Printf("找到 %d 个结果:\n", len(packages))
	for i, pkg := range packages {
		if i >= 5 {
			fmt.Println("... 更多结果省略 ...")
			break
		}
		fmt.Printf("%d. %s (版本: %s, 下载量: %d)\n", i+1, pkg.Name, pkg.Version, pkg.Downloads)
	}
	fmt.Println()
}

func getVersions(ctx context.Context) {
	fmt.Println("=== 获取版本列表 ===")
	repo := repository.NewRepository()

	versions, err := repo.GetGemVersions(ctx, "rails")
	if err != nil {
		fmt.Printf("获取版本失败: %v\n", err)
		return
	}

	fmt.Printf("rails 共有 %d 个版本:\n", len(versions))
	for i, ver := range versions {
		if i >= 5 {
			fmt.Println("... 更多版本省略 ...")
			break
		}
		fmt.Printf("%d. %s (下载量: %d, 发布时间: %s)\n",
			i+1, ver.Number, ver.DownloadsCount, ver.CreatedAt.Format("2006-01-02"))
	}
	fmt.Println()
}

func getDownloadStats(ctx context.Context) {
	fmt.Println("=== 下载统计 ===")
	repo := repository.NewRepository()

	// 获取仓库总下载量
	repoStats, err := repo.Downloads(ctx)
	if err != nil {
		fmt.Printf("获取下载统计失败: %v\n", err)
		return
	}

	fmt.Printf("RubyGems 仓库总下载量: %d\n", repoStats.TotalDownloads)

	// 获取特定版本下载量
	verStats, err := repo.VersionDownloads(ctx, "rails", "7.0.5")
	if err != nil {
		fmt.Printf("获取版本下载统计失败: %v\n", err)
		return
	}

	fmt.Printf("rails 7.0.5 版本下载量: %d\n", verStats.VersionDownloads)
	fmt.Printf("rails 总下载量: %d\n", verStats.TotalDownloads)
	fmt.Println()
}
