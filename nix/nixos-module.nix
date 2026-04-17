{ self }:
{ config, lib, pkgs, ... }:
let
  cfg = config.services.ghmd;
  common = import ./common.nix { inherit self; } { inherit lib pkgs; };
  startScript = common.mkStartScript cfg;
in
{
  options.services.ghmd = common.mkCommonOptions {
    rootDirType = lib.types.path;
    hasRootDirDefault = false;
    rootDirExample = "/var/lib/ghmd";
    rootDirDescription = "Root directory to serve.";
  } // {
    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Open the configured TCP port in the firewall.";
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.ghmd = {
      description = "ghmd markdown server";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        ExecStart = "${startScript}/bin/ghmd-server";
        WorkingDirectory = cfg.rootDir;
        DynamicUser = true;
        Restart = "on-failure";
        RestartSec = 5;
        ProtectSystem = "strict";
        ProtectHome = true;
        BindReadOnlyPaths = [ cfg.rootDir ];
        PrivateTmp = true;
        NoNewPrivileges = true;
        LockPersonality = true;
        RestrictSUIDSGID = true;
        RestrictRealtime = true;
        SystemCallArchitectures = "native";
      };
    };

    networking.firewall.allowedTCPPorts = lib.mkIf cfg.openFirewall [ cfg.port ];
  };
}
