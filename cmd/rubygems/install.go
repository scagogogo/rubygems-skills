package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/scagogogo/rubygems-skills/pkg/install"
)

func installCmd() *cobra.Command {
	opts := install.NewInstallOptions()
	var force, noDev, noBundler, noUpdate, noSudo bool

	c := &cobra.Command{
		Use:   "install",
		Short: "Auto-install Ruby/RubyGems on the current host (apt/yum/dnf/apk/pacman/brew/choco/scoop/zypper)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.
				WithForceReinstall(force).
				WithDevHeaders(!noDev).
				WithBundler(!noBundler).
				WithUpdatePackageIndex(!noUpdate).
				WithSudo(!noSudo)

			installer := install.NewInstaller(opts)

			platform, err := installer.DetectPlatform()
			if err != nil {
				return fmt.Errorf("platform detection failed: %w", err)
			}
			fmt.Printf("Detected platform: %s\n", platform)

			installed, info, _ := installer.IsInstalled()
			if installed && !force {
				fmt.Printf("Ruby already installed: %s (gem: %s). Use --force to reinstall.\n", info.RubyVersion, info.GemVersion)
				return nil
			}
			if installed && force {
				fmt.Printf("Ruby already installed: %s, force reinstalling...\n", info.RubyVersion)
			}

			fmt.Println("Installing Ruby/RubyGems...")
			ctx, cancel := ctxWithTimeout()
			defer cancel()
			result, err := installer.Install(ctx)
			if err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}
			if result.AlreadyInstalled {
				fmt.Printf("Ruby already installed: %s (gem: %s)\n", result.RubyVersion, result.GemVersion)
				return nil
			}
			fmt.Println("Installation successful!")
			fmt.Printf("  Ruby version: %s\n", result.RubyVersion)
			fmt.Printf("  gem version: %s\n", result.GemVersion)
			fmt.Printf("  Ruby path: %s\n", result.RubyPath)
			fmt.Printf("  gem path: %s\n", result.GemPath)
			fmt.Printf("  Package manager: %s\n", result.PackageManager)
			if len(result.CommandsRun) > 0 {
				fmt.Println("  Commands executed:")
				for _, cmd := range result.CommandsRun {
					fmt.Printf("    %s\n", cmd)
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "Force reinstall")
	c.Flags().BoolVar(&noDev, "no-dev", false, "Skip dev headers (ruby-dev/ruby-devel)")
	c.Flags().BoolVar(&noBundler, "no-bundler", false, "Skip bundler")
	c.Flags().BoolVar(&noUpdate, "no-update", false, "Skip package index update")
	c.Flags().BoolVar(&noSudo, "no-sudo", false, "Don't use sudo")
	return c
}

func platformCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "platform",
		Short: "Detect and print the current platform (OS, distro, package manager)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			installer := install.NewInstaller()
			platform, err := installer.DetectPlatform()
			if err != nil {
				return fmt.Errorf("platform detection failed: %w", err)
			}
			if flagJSON {
				printOutput(platform)
				return nil
			}
			fmt.Println(platform)
			return nil
		},
	}
}
