{
  description = "wtc - Worktree cleanup tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];

      perSystem = {
        pkgs,
        self',
        ...
      }: let
        # nixpkgs' nilaway (2025-03-07) vendors an x/tools whose go/ssa panics
        # under Go 1.27. Track upstream main until nixpkgs ships a Go 1.27-ready
        # build, then drop this override. doCheck is off because upstream's own
        # test suite is OOM-killed on 7 GB CI runners.
        nilaway = pkgs.nilaway.overrideAttrs (_: {
          version = "0-unstable-2026-11-22";
          src = pkgs.fetchFromGitHub {
            owner = "uber-go";
            repo = "nilaway";
            rev = "7f797c9f2b67a10d81b207ea8c29241af77f85d7";
            hash = "sha256-H9Ex0dNdDTm5gHUeUMBVL8/02qW1IuHzTRLAgeh9/AU=";
          };
          doCheck = false;
          vendorHash = "sha256-qVmvDneq6V/q5UHZ/Cjjqd5/XPPNfvVGoxwg9nz4/Ds=";
        });
      in {
        packages.default = pkgs.buildGoModule {
          pname = "wtc";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-6yVgZr3+5bPP5AuMmudMl25eUzlLGJorQ7M+4jpy+hY=";
          ldflags = ["-s" "-w"];
          subPackages = ["cmd/wtc"];
          meta = {
            description = "Worktree cleanup tool with TUI explorer";
            mainProgram = "wtc";
          };
        };

        checks.gate = self'.packages.default.overrideAttrs (old: {
          pname = "wtc-gate";
          nativeBuildInputs = old.nativeBuildInputs ++ [pkgs.git pkgs.golangci-lint nilaway];
          doCheck = false;
          buildPhase = ''
            runHook preBuild
            export HOME=$TMPDIR
            export GOLANGCI_LINT_CACHE=$TMPDIR/golangci-lint
            golangci-lint run ./...
            nilaway -include-pkgs=github.com/noamsto/wt ./...
            go test -race ./...
            runHook postBuild
          '';
          installPhase = "touch $out";
        });
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
