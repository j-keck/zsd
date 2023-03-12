{
  description = "A simple Go package";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "github:Nixos/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let

      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 lastModifiedDate;

      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

    in
    {

      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          zsd = pkgs.buildGoModule {
            pname = "zsd";
            inherit version;
            # In 'nix develop', we don't need a copy of the source tree
            # in the Nix store.
            src = ./.;

            #vendorSha256 = pkgs.lib.fakeSha256;
            vendorSha256 = "sha256-9/E6yH6m4mioeuntA87jbgh6aMZfclJy3IAb/+P490Y=";
          };
        });

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.zsd}/bin/zsd";
        };
      });

      formatter = forAllSystems (system: nixpkgsFor.${system}.nixpkgs-fmt);

      devShells = forAllSystems (system: {
        default = nixpkgsFor.${system}.mkShell {
          packages = [
            nixpkgsFor.${system}.go
            self.packages.${system}.zsd
          ];
        };
      });


      defaultPackage = forAllSystems (system: self.packages.${system}.zsd);
    };
}
