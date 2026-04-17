{ self }:
{ config, lib, pkgs, ... }:
let
  cfg = config.programs.ghmd;
  package = cfg.package;
  startScript = pkgs.writeShellScriptBin "ghmd-server" ''
    set -eu
    args=()
    ${lib.optionalString (cfg.rootDir != null) ''args+=(-server ${lib.escapeShellArg cfg.rootDir})''}
    args+=(-host ${lib.escapeShellArg cfg.host})
    args+=(-port ${toString cfg.port})
    args+=(-theme ${lib.escapeShellArg cfg.theme})
    exec ${lib.getExe package} "''${args[@]}"
  '';
in
{
  options.programs.ghmd = {
    enable = lib.mkEnableOption "ghmd server service";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.system}.default;
      defaultText = lib.literalExpression "self.packages.${pkgs.system}.default";
      description = "ghmd package to run.";
    };

    rootDir = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/home/alice/docs";
      description = "Root directory to serve. Null means current working directory.";
    };

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

  config = lib.mkIf cfg.enable {
    systemd.user.services.ghmd = {
      Unit = {
        Description = "ghmd markdown server";
        After = [ "network.target" ];
      };

      Service = {
        ExecStart = "${startScript}/bin/ghmd-server";
        Restart = "on-failure";
        RestartSec = 5;
      };

      Install = {
        WantedBy = [ "default.target" ];
      };
    };
  };
}
