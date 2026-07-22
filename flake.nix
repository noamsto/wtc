{
  description = "wtc - Worktree cleanup tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];

      perSystem = {pkgs, ...}: {
        packages.default = pkgs.buildGoModule {
          pname = "wtc";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-2ucj2nIyv8s7Bc/1nE9yEcS4X+ce3SZIEs/iL/FEF+8=";
          ldflags = ["-s" "-w"];
          subPackages = ["cmd/wtc"];
          meta = {
            description = "Worktree cleanup tool with TUI explorer";
            mainProgram = "wtc";
          };
        };
      };

      flake = {
        homeManagerModules.default = {
          config,
          lib,
          pkgs,
          ...
        }: let
          cfg = config.programs.wtc;
          wtcPkg = inputs.self.packages.${pkgs.system}.default;
        in {
          options.programs.wtc = {
            enable = lib.mkEnableOption "wtc - worktree cleanup tool";
          };

          config = lib.mkIf cfg.enable {
            home.packages = [wtcPkg];

            xdg.configFile."fish/completions/wtc.fish".source = ./completions/wtc.fish;
          };
        };
      };
    };
}
