{ goos ? "linux", with-dev-tools ? false }:
let

  fetchNixpkgs = {rev, sha256}: builtins.fetchTarball {
    url = "https://github.com/NixOS/nixpkgs-channels/archive/${rev}.tar.gz";
    inherit sha256;
  };

  pkgs = import (fetchNixpkgs {
    rev = "8a9807f1941d046f120552b879cf54a94fca4b38";
    sha256 = "0s8gj8b7y1w53ak138f3hw1fvmk40hkpzgww96qrsgf490msk236";
  }) {};

  version =
    let lookup-version = pkgs.stdenv.mkDerivation {
          src = builtins.path { name = "git"; path = ./.git; };
          name = "zsd-lookup-version";
          phases = "buildPhase";
          buildPhase = ''
            mkdir -p $out
            ${pkgs.git}/bin/git --git-dir=$src describe --always --tags > $out/version
          '';
        };
    in pkgs.lib.removeSuffix "\n" (builtins.readFile "${lookup-version}/version");


  zsd = pkgs.buildGo112Module rec {
    pname = "zsd";
    inherit version;
    src = pkgs.nix-gitignore.gitignoreSource [ ".gitignore" ] ./.;
    modSha256 = "1i8bd6cli45zcl40xc5cw0ywj8nz1xnz5jyxq5rd7adw89m0fd40";

    preBuild = ''
      export GOOS=${goos}
    '';

    CGO_ENABLED = 0;

    buildFlagsArray = ''
      -ldflags=
      -X main.version=${version}
    '';

    installPhase = ''
      mkdir -p $out

      BIN_PATH=${if goos == pkgs.stdenv.buildPlatform.parsed.kernel.name
                 then "$GOPATH/bin"
                 else "$GOPATH/bin/${goos}_$GOARCH"}

      mkdir -p $out/bin
      cp $BIN_PATH/zsd $out/bin

      mkdir -p $out/share
      cp LICENSE $out/share
    '';
  };



  site =
    let theme = pkgs.fetchFromGitHub {
          owner = "alex-shpak";
          repo = "hugo-book";
          rev = "dae803fa442973561821a44b08e3a964614d07df";
          sha256 = "0dpb860kddclsqnr4ls356jn4d1l8ymw5rs9wfz2xq4kkrgls4dl";
        };
    in pkgs.stdenv.mkDerivation rec {
      name = "zsd-site";
      inherit version;
      src = ./doc/site;
      buildPhase = ''
        cp -a ${theme}/. themes/book
        ${pkgs.hugo}/bin/hugo --minify
      '';
      installPhase = ''
        cp -r public $out
      '';
    };

in

if pkgs.lib.inNixShell then pkgs.mkShell {
  buildInputs = with pkgs;
    [ go_1_12 ] ++
    (if with-dev-tools
     then [ hugo ]
     else []);

  shellHooks = ''
    unset GOPATH
  '';
}
else {
  inherit zsd site;
}
