{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {

  buildInputs = [
    pkgs.vgo2nix
    pkgs.go
  ];

  shellHook = ''
    unset GOPATH
  '';

}
