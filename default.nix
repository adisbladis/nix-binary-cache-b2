{ stdenv, buildGoPackage }:

buildGoPackage rec {
  name = "nix-binary-cache-b2-${version}";
  version = "0.0.1";
  goPackagePath = "github.com/adisbladis/nix-binary-cache-b2";

  src = ./.;
  goDeps = ./deps.nix;

  CGO_ENABLED = 0;

  meta = with stdenv.lib; {
    description = "Proxy backblaze b2";
  };
}
