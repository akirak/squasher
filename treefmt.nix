{
  projectRootFile = "treefmt.nix";

  # See https://github.com/numtide/treefmt-nix#supported-programs

  programs.gofmt.enable = true;

  # JSON
  programs.biome.enable = true;
  programs.biome = {
    includes = [ "renovate.json" ];
    settings = {
      json = {
        formatter = {
          indentStyle = "space";
          indentWidth = 2;
          lineEnding = "lf";
        };
      };
    };
  };

  # GitHub Actions
  programs.yamlfmt.enable = true;
  programs.actionlint.enable = true;

  # Markdown
  programs.mdformat.enable = true;

  # Nix
  programs.nixfmt.enable = true;
}
