package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
	// 创建一个基础仓库
	baseRepo := repository.NewRepository()

	// 创建一个内存缓存
	memCache := cache.NewMemoryCache(5*time.Minute, 15*time.Minute)

	// 创建一个带缓存的仓库，缓存时间为5分钟
	cachedRepo := repository.NewCachedRepository(baseRepo, 5*time.Minute, memCache)

	// 创建一个上下文
	ctx := context.Background()

	// 首次查询，会从API获取数据
	fmt.Println("===== 首次查询 (从API获取) =====")
	start := time.Now()
	pkg, err := cachedRepo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}
	fmt.Printf("查询耗时: %v\n", time.Since(start))
	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Printf("缓存项数量: %d\n", cachedRepo.GetCacheStats())

	// 再次查询相同的包，会从缓存获取
	fmt.Println("\n===== 再次查询 (从缓存获取) =====")
	start = time.Now()
	pkg, err = cachedRepo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}
	fmt.Printf("查询耗时: %v\n", time.Since(start))
	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Printf("缓存项数量: %d\n", cachedRepo.GetCacheStats())

	// 查询另一个包，会从API获取
	fmt.Println("\n===== 查询另一个包 (从API获取) =====")
	start = time.Now()
	pkg, err = cachedRepo.GetPackage(ctx, "rake")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}
	fmt.Printf("查询耗时: %v\n", time.Since(start))
	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Printf("缓存项数量: %d\n", cachedRepo.GetCacheStats())

	// 使用自定义缓存
	fmt.Println("\n===== 使用自定义缓存 =====")
	customCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
	customCachedRepo := repository.NewCachedRepository(baseRepo, 10*time.Minute, customCache)

	pkg, err = customCachedRepo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("获取包信息失败: %v\n", err)
		return
	}
	fmt.Printf("包名: %s\n", pkg.Name)
	fmt.Printf("版本: %s\n", pkg.Version)
	fmt.Printf("缓存项数量: %d\n", customCachedRepo.GetCacheStats())

	// 清空缓存
	fmt.Println("\n===== 清空缓存 =====")
	cachedRepo.ClearCache()
	fmt.Printf("清空后缓存项数量: %d\n", cachedRepo.GetCacheStats())

	// 关闭缓存
	fmt.Println("\n===== 关闭缓存 =====")
	cachedRepo.Close()
	customCachedRepo.Close()

	/*
		示例输出：
		===== 首次查询 (从API获取) =====
		查询耗时: 1.234567s
		包名: rails
		版本: 7.0.5
		缓存项数量: 1

		===== 再次查询 (从缓存获取) =====
		查询耗时: 123µs
		包名: rails
		版本: 7.0.5
		缓存项数量: 1

		===== 查询另一个包 (从API获取) =====
		查询耗时: 987.654ms
		包名: rake
		版本: 13.0.6
		缓存项数量: 2

		===== 使用自定义缓存 =====
		包名: rails
		版本: 7.0.5
		缓存项数量: 1

		===== 清空缓存 =====
		清空后缓存项数量: 0

		===== 关闭缓存 =====
	*/
}
