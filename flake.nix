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
      }: {
        packages.default = pkgs.buildGoModule {
          pname = "wtc";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-Nyj2oXdHk+ZhmUUky22YGtbFYyyJIyl5E4ky8FGVYXE=";
          ldflags = ["-s" "-w"];
          subPackages = ["cmd/wtc"];
          meta = {
            description = "Worktree cleanup tool with TUI explorer";
            mainProgram = "wtc";
          };
        };

        checks.gate = self'.packages.default.overrideAttrs (old: {
          pname = "wtc-gate";
          nativeBuildInputs = old.nativeBuildInputs ++ [pkgs.git pkgs.golangci-lint pkgs.nilaway];
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
