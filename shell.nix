{ pkgs ? import <nixpkgs> {} }:
  pkgs.mkShell {
      buildInputs = with pkgs; [
          go
          gosec
          golint
          errcheck
          git
          ffmpeg
      ];
  }