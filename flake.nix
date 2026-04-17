{
  description = "ghmd - Markdown to HTML renderer based on Goldmark";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    {
      homeManagerModules.default = import ./nix/home-manager.nix { inherit self; };
    } // flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        lib = pkgs.lib;
      in
      {
        packages.ghmd = pkgs.buildGoModule {
          pname = "ghmd";
          version = "0.1.0";
          src = self;
          vendorHash = "sha256-U4k4+jSmz3rnm6+gfSqHSTPM1uVcOSKf1m/u2elqfsc=";
        };

        packages.default = self.packages.${system}.ghmd;

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.ghmd;
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go ];
        };
      }
    );
}
