{ self }:
{
  lib,
  pkgs,
}:
let
  defaultPackage = self.packages.${pkgs.system}.default;
in
{
  mkStartScript = cfg:
    pkgs.writeShellScriptBin "ghmd-server" ''
      set -eu
      args=()
      args+=(-server ${lib.escapeShellArg (toString cfg.rootDir)})
      args+=(-host ${lib.escapeShellArg cfg.host})
      args+=(-port ${toString cfg.port})
      args+=(-theme ${lib.escapeShellArg cfg.theme})
      exec ${lib.getExe cfg.package} "''${args[@]}"
    '';

  mkCommonOptions = {
    rootDirType,
    rootDirDefault ? null,
    hasRootDirDefault ? true,
    rootDirExample,
    rootDirDescription,
  }: {
    enable = lib.mkEnableOption "ghmd server service";

    package = lib.mkOption {
      type = lib.types.package;
      default = defaultPackage;
      defaultText = lib.literalExpression "self.packages.${pkgs.system}.default";
      description = "ghmd package to run.";
    };

    rootDir = lib.mkOption ({
      type = rootDirType;
      example = rootDirExample;
      description = rootDirDescription;
    } // lib.optionalAttrs hasRootDirDefault {
      default = rootDirDefault;
    });

    host = lib.mkOption {
      type = lib.types.str;
      default = "127.0.0.1";
      description = "Listen host.";
    };

    port = lib.mkOption {
      type = lib.types.port;
      default = 8080;
      description = "Listen port.";
    };

    theme = lib.mkOption {
      type = lib.types.str;
      default = "github";
      description = "Code highlighting theme.";
    };
  };
}
