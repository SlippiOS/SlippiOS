{ config, lib, cachyosKernels, ... }:

let
  cfg = config.slippios.kernel;
in {
  options.slippios.kernel = {
    enable = lib.mkEnableOption "the CachyOS gaming kernel";

    variant = lib.mkOption {
      type = lib.types.enum [
        "lts"
        "latest"
        "bore"
        "lts-lto"
        "latest-lto"
        "bore-lto"
      ];
      default = "lts";
      description = ''
        CachyOS kernel variant. lts stays on the 6.18 series; bore uses the
        latest kernel with the BORE scheduler. The -lto variants use Clang
        ThinLTO and are usually larger.
      '';
    };

    schedExt = lib.mkOption {
      type = lib.types.nullOr (lib.types.enum [
        "scx_lavd"
        "scx_bpfland"
        "scx_rustland"
      ]);
      default = null;
      description = ''
        Optional sched-ext scheduler to run instead of the kernel's built-in
        scheduler. The CachyOS kernel enables CONFIG_SCHED_CLASS_EXT.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    boot.kernelPackages = cachyosKernels."linuxPackages-cachyos-${cfg.variant}";

    services.scx = lib.mkIf (cfg.schedExt != null) {
      enable = true;
      scheduler = cfg.schedExt;
    };
  };
}
