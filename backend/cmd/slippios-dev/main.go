package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SlippiOS/SlippiOS/backend/internal/host"
	"github.com/spf13/cobra"
)

func main() {
	if filepath.Base(os.Args[0]) == "slippios-flash" {
		if err := host.RunFlashHelper(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}
	if err := newRootCommand().Execute(); err != nil {
		if errors.Is(err, host.ErrCancelled) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "slippios-dev",
		Short:         "SlippiOS developer and release tools",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("command required")
		},
	}
	root.AddCommand(
		newBuildCommand(),
		newDevCommand(),
		newFlashCommand(),
		newRunCommand(),
		newCleanCommand(),
		newFetchCommand(),
		newSignCommand(),
		newReleaseCommand(),
		newPublishCommand(),
		newTestCommand(),
		newNixCommand(),
	)
	return root
}

func newBuildCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "build",
		Short: "Build images",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("build action required")
		},
	}
	var opts host.BuildOptions
	image := &cobra.Command{
		Use:   "image",
		Short: "Build a USB image locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.BuildImage(ws, opts)
		},
	}
	image.Flags().BoolVar(&opts.Dev, "dev", false, "build the uncompressed dev image")
	image.Flags().BoolVar(&opts.Minimal, "minimal", false, "build the minimal image")
	image.Flags().StringVar(&opts.Config, "config", "", "NixOS configuration name")
	image.Flags().StringVar(&opts.Result, "result", "", "result link (default result-local-image)")
	image.Flags().IntVar(&opts.Jobs, "jobs", 0, "Nix build cores (default: min(nproc, 8))")
	image.Flags().BoolVar(&opts.SkipClean, "skip-clean", false, "skip build history cleanup")
	image.Flags().StringVar(&opts.TmpDir, "tmpdir", "", "build TMPDIR (default .cache/build-tmp)")
	command.AddCommand(image)
	return command
}

func newDevCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "dev",
		Short: "Local development sandbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("dev action required")
		},
	}
	command.AddCommand(&cobra.Command{
		Use:                "cli [args...]",
		Short:              "Run the CLI against the local sandbox",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.DevCLI(ws, args)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "ssh",
		Short: "SSH into the running dev VM",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.DevSSH(ws)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "push",
		Short: "Push the CLI and exported shell to the dev VM",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.DevPush(ws)
		},
	})
	return command
}

func newFlashCommand() *cobra.Command {
	var opts host.FlashOptions
	command := &cobra.Command{
		Use:   "flash",
		Short: "Detect USB media, confirm, and flash the newest dev image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.FlashInteractive(ws, opts)
		},
	}
	command.Flags().StringVar(&opts.Device, "device", "", "skip detection and use this device")
	command.Flags().StringVar(&opts.ImageDir, "image-dir", "", "signed image directory")
	command.Flags().BoolVar(&opts.Yes, "yes", false, "skip the confirmation")
	command.Flags().BoolVar(&opts.DryRun, "dry-run", false, "show the target and exit")
	command.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the helper, device, image, and USB media",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.FlashStatus(ws)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Remove the passwordless flash helper",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.FlashDisable(ws)
		},
	})
	var enableOpts host.FlashOptions
	enable := &cobra.Command{
		Use:    "enable",
		Short:  "Install the root flash helper",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.FlashEnable(ws, enableOpts)
		},
	}
	enable.Flags().StringVar(&enableOpts.Device, "device", "", "target device")
	enable.Flags().StringVar(&enableOpts.ImageDir, "image-dir", "", "signed image directory")
	enable.Flags().BoolVar(&enableOpts.AllowFixed, "allow-fixed", false, "allow a USB NVMe enclosure")
	command.AddCommand(enable)
	return command
}

func newRunCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "run",
		Short: "Run a local image in QEMU",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("run action required")
		},
	}
	var opts host.RunOptions
	image := &cobra.Command{
		Use:   "image [IMAGE]",
		Short: "Boot a local image under OVMF/QEMU",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				opts.Image = args[0]
			}
			return host.RunImage(ws, opts)
		},
	}
	image.Flags().StringVar(&opts.DiskSize, "disk-size", "", "virtual disk size (default 16G)")
	image.Flags().StringVar(&opts.DiskMode, "disk", "", "virtio (default) or usb")
	image.Flags().StringVar(&opts.Display, "display", "", "QEMU display backend (default gtk)")
	image.Flags().StringVar(&opts.VGA, "vga", "", "std (default) or virtio")
	image.Flags().BoolVar(&opts.SecureBoot, "secure-boot", false, "use OVMF Secure Boot")
	image.Flags().StringVar(&opts.SSHPort, "ssh-port", "", "host SSH port (default 2222)")
	image.Flags().StringVar(&opts.Serial, "serial", "", "write the serial console to this file")
	image.Flags().StringVar(&opts.Monitor, "monitor", "", "QEMU monitor socket")
	command.AddCommand(image)
	command.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Power off the running QEMU guest cleanly",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.StopImage(ws)
		},
	})
	return command
}

func newCleanCommand() *cobra.Command {
	var opts host.CleanOptions
	command := &cobra.Command{
		Use:   "clean",
		Short: "Remove disposable cache and build history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.Clean(ws, opts)
		},
	}
	command.Flags().BoolVar(&opts.All, "all", false, "clean the VM cache and build history")
	command.Flags().BoolVar(&opts.History, "history", false, "clean build history only")
	command.Flags().BoolVar(&opts.Hard, "hard", false, "wipe .cache and the nix-portable store")
	return command
}

func newFetchCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "fetch",
		Short: "Download published images",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("fetch action required")
		},
	}
	var opts host.FetchOptions
	image := &cobra.Command{
		Use:   "image",
		Short: "Download and boot the newest published USB image",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			if opts.DiskSize == "" {
				opts.DiskSize = envOr("SLIPPI_DISK_SIZE", "16G")
			}
			return host.FetchImage(ws, opts)
		},
	}
	image.Flags().BoolVar(&opts.Offline, "offline", false, "use only a verified cached image")
	image.Flags().BoolVar(&opts.DryRun, "dry-run", false, "show release metadata without downloading")
	image.Flags().BoolVar(&opts.NoRun, "no-run", false, "download and verify without booting")
	image.Flags().StringVar(&opts.DiskSize, "disk-size", "", "virtual disk size (default 16G)")
	command.AddCommand(image)
	return command
}

func newSignCommand() *cobra.Command {
	var inShell bool
	command := &cobra.Command{
		Use:   "sign",
		Short: "Sign images for Secure Boot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("sign action required")
		},
	}
	command.PersistentFlags().BoolVar(&inShell, "in-shell", false, "internal: tools are already in PATH")
	command.PersistentFlags().MarkHidden("in-shell")
	command.AddCommand(&cobra.Command{
		Use:   "image <unsigned.raw> <signed.raw>",
		Short: "Sign a SlippiOS image",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.SignImage(ws, host.SignOptions{Input: args[0], Output: args[1], InShell: inShell})
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "uki <unsigned.efi> <signed.efi>",
		Short: "Sign a standalone UKI",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.SignUKI(ws, host.SignOptions{Input: args[0], Output: args[1], InShell: inShell})
		},
	})
	return command
}

func newReleaseCommand() *cobra.Command {
	var inShell bool
	command := &cobra.Command{
		Use:   "release",
		Short: "Build, sign, and publish a USB release",
		Args:  cobra.NoArgs,
	}
	command.PersistentFlags().BoolVar(&inShell, "in-shell", false, "internal: tools are already in PATH")
	command.PersistentFlags().MarkHidden("in-shell")

	var full host.ReleaseOptions
	command.RunE = func(cmd *cobra.Command, args []string) error {
		ws, err := host.FindWorkspace()
		if err != nil {
			return err
		}
		full.InShell = inShell
		return host.Release(ws, full)
	}
	command.Flags().BoolVar(&full.DryRun, "dry-run", false, "generate assets without uploading")
	command.Flags().BoolVar(&full.AllowDirty, "allow-dirty", false, "allow a dirty working tree")
	command.Flags().BoolVar(&full.NoBuild, "no-build", false, "use the existing result-local-image")
	command.Flags().BoolVar(&full.Publish, "publish", false, "publish the release after uploading")
	command.Flags().BoolVar(&full.Force, "force", false, "replace assets on an existing release")
	command.Flags().StringVar(&full.Tag, "tag", "", "release tag (default v<version>)")
	command.Flags().StringVar(&full.AssetsDir, "assets-dir", "", "output directory")
	command.Flags().BoolVar(&full.KeepRaw, "keep-raw", false, "keep the signed raw image")
	command.Flags().StringVar(&full.Result, "result", "", "result link (default result-local-image)")

	var dev host.ReleaseDevOptions
	devCommand := &cobra.Command{
		Use:   "dev",
		Short: "Build, sign, and publish a dev update",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			dev.InShell = inShell
			return host.ReleaseDev(ws, dev)
		},
	}
	devCommand.Flags().BoolVar(&dev.DryRun, "dry-run", false, "generate assets without uploading")
	devCommand.Flags().BoolVar(&dev.AllowDirty, "allow-dirty", false, "allow a dirty working tree")
	devCommand.Flags().BoolVar(&dev.NoBuild, "no-build", false, "use the existing result-local-image-dev")
	devCommand.Flags().BoolVar(&dev.KeepRaw, "keep-raw", false, "keep the signed full image for the initial flash")
	devCommand.Flags().StringVar(&dev.Result, "result", "", "result link (default result-local-image-dev)")
	devCommand.Flags().StringVar(&dev.AssetsDir, "assets-dir", "", "output directory")
	command.AddCommand(devCommand)

	var minimal host.ReleaseMinimalOptions
	minimalCommand := &cobra.Command{
		Use:   "minimal",
		Short: "Build and add the minimal image to an existing release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			minimal.InShell = inShell
			return host.ReleaseMinimal(ws, minimal)
		},
	}
	minimalCommand.Flags().StringVar(&minimal.AssetsDir, "assets-dir", "", "release assets directory")
	command.AddCommand(minimalCommand)
	return command
}

func newPublishCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "publish",
		Short: "Publish release assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("publish action required")
		},
	}
	var r2opts host.PublishR2Options
	r2 := &cobra.Command{
		Use:   "r2 <assets-dir>",
		Short: "Publish update assets to Cloudflare R2",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.PublishR2(ws, args[0], r2opts)
		},
	}
	r2.Flags().BoolVar(&r2opts.Dev, "dev", false, "publish to the dev channel")
	r2.Flags().BoolVar(&r2opts.DryRun, "dry-run", false, "show usage without uploading")
	command.AddCommand(r2)

	sourceforge := &cobra.Command{
		Use:   "sourceforge <assets-dir> [user]",
		Short: "Upload full images to SourceForge",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			user := ""
			if len(args) == 2 {
				user = args[1]
			}
			return host.PublishSourceForge(ws, args[0], user)
		},
	}
	command.AddCommand(sourceforge)
	return command
}

func newTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Run Go tests and the headless Godot parse",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			return host.RunTests(ws)
		},
	}
}

func newNixCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "nix [args...]",
		Short:              "Run Nix, or open a shell with the project tools",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := host.FindWorkspace()
			if err != nil {
				return err
			}
			nix := host.NewNix(ws)
			if len(args) > 0 {
				return nix.Exec(args...)
			}
			return nix.Exec("shell", "--inputs-from", ".",
				"nixpkgs#godot_4", "nixpkgs#qemu", "nixpkgs#zsync",
				"nixpkgs#xz", "nixpkgs#jq")
		},
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
