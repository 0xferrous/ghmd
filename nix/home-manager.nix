{ self }:
{ config, lib, pkgs, ... }:
let
  cfg = config.programs.ghmd;
  common = import ./common.nix { inherit self; } { inherit lib pkgs; };
  startScript = common.mkStartScript cfg;
in
{
  options.programs.ghmd = common.mkCommonOptions {
    rootDirType = lib.types.str;
    rootDirDefault = ".";
    rootDirExample = "/home/alice/docs";
    rootDirDescription = "Root directory to serve.";
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
