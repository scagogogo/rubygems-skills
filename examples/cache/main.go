package main

import (
	"context"
	"fmt"
	"time"

	"github.com/scagogogo/rubygems-skills/pkg/cache"
	"github.com/scagogogo/rubygems-skills/pkg/repository"
)

func main() {
	// create a base repository
	baseRepo := repository.NewRepository()

	// create a memory cache
	memCache := cache.NewMemoryCache(5*time.Minute, 15*time.Minute)

	// create a cached repository, cache time is 5 minutes
	cachedRepo := repository.NewCachedRepository(baseRepo, 5*time.Minute, memCache)

	// create a context
	ctx := context.Background()

	// first query, fetches from API
	fmt.Println("===== first query (from API) =====")
	start := time.Now()
	pkg, err := cachedRepo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}
	fmt.Printf("query elapsed: %v\n", time.Since(start))
	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Printf("cache count: %d\n", cachedRepo.GetCacheStats())

	// query same package again, fetches from cache
	fmt.Println("\n===== query again (from cache) =====")
	start = time.Now()
	pkg, err = cachedRepo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}
	fmt.Printf("query elapsed: %v\n", time.Since(start))
	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Printf("cache count: %d\n", cachedRepo.GetCacheStats())

	// query another package, fetches from API
	fmt.Println("\n===== query another package (from API) =====")
	start = time.Now()
	pkg, err = cachedRepo.GetPackage(ctx, "rake")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}
	fmt.Printf("query elapsed: %v\n", time.Since(start))
	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Printf("cache count: %d\n", cachedRepo.GetCacheStats())

	// use custom cache
	fmt.Println("\n===== use custom cache =====")
	customCache := cache.NewMemoryCache(10*time.Minute, 30*time.Minute)
	customCachedRepo := repository.NewCachedRepository(baseRepo, 10*time.Minute, customCache)

	pkg, err = customCachedRepo.GetPackage(ctx, "rails")
	if err != nil {
		fmt.Printf("failed to get package info: %v\n", err)
		return
	}
	fmt.Printf("package name: %s\n", pkg.Name)
	fmt.Printf("version: %s\n", pkg.Version)
	fmt.Printf("cache count: %d\n", customCachedRepo.GetCacheStats())

	// clear cache
	fmt.Println("\n===== clear cache =====")
	cachedRepo.ClearCache()
	fmt.Printf("cache count after clear: %d\n", cachedRepo.GetCacheStats())

	// close cache
	fmt.Println("\n===== close cache =====")
	cachedRepo.Close()
	customCachedRepo.Close()

	/*
		example output:
		===== first query (from API) =====
		query elapsed: 1.234567s
		package name: rails
		version: 7.0.5
		cache count: 1

		===== query again (from cache) =====
		query elapsed: 123µs
		package name: rails
		version: 7.0.5
		cache count: 1

		===== query another package (from API) =====
		query elapsed: 987.654ms
		package name: rake
		version: 13.0.6
		cache count: 2

		===== use custom cache =====
		package name: rails
		version: 7.0.5
		cache count: 1

		===== clear cache =====
		cache count after clear: 0

		===== close cache =====
	*/
}
