{
  description = "Squash Git commits";

  inputs.systems.url = "github:nix-systems/default";

  nixConfig = {
    extra-substituters = [
      "https://akirak.cachix.org"
    ];
    extra-trusted-public-keys = [
      "akirak.cachix.org-1:WJrEMdV1dYyALkOdp/kAECVZ6nAODY5URN05ITFHC+M="
    ];
  };

  outputs =
    {
      self,
      nixpkgs,
      systems,
    }:
    let

      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs (import systems);

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

    in
    {

      # Provide some binary packages for selected system types.
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        rec {
          default = self.packages.${system}.squasher;

          squasher = pkgs.buildGoModule {
            pname = "squasher";
            inherit version;
            src = self.outPath;

            # Tests fail due to missing a git binary
            doCheck = false;

            # This hash locks the dependencies of this package. It is
            # necessary because of how Go requires network access to resolve
            # VCS.  See https://www.tweag.io/blog/2021-03-04-gomod2nix/ for
            # details. Normally one can build with a fake sha256 and rely on native Go
            # mechanisms to tell you what the hash should be or determine what
            # it should be "out-of-band" with other tooling (eg. gomod2nix).
            # To begin with it is recommended to set this, but one must
            # remeber to bump this hash when your dependencies change.
            # vendorSha256 = pkgs.lib.fakeSha256;
            # vendorSha256 = null;

            vendorSha256 = "sha256-BdRC0HuyxnSInK3HqzLD3Q53VR0nS+QzfD0RmwmBwJI=";
          };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = [
              pkgs.go
              pkgs.gopls
            ];
          };
        }
      );

    };
}
