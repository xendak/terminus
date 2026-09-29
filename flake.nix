{
  description = "Terminus MVP — Go + PostgreSQL devshell";

  # Pinned nixpkgs, verified 2026-09-28 (plans/mvp/notes.md "Environment"):
  # go 1.26.7, postgresql_18 18.6, plantuml 1.2026.6.
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/83199d0d373dd3ac2b9a1996b1d0263f76ab7a4c";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system}; in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go            # 1.26 toolchain: build, vet, test
              gopls         # language server
              postgresql_18 # initdb, pg_ctl, psql, createdb, pg_isready
              plantuml      # diagram re-renders (docs/especificacao/diagrams)
            ];
          };
        });
    };
}
