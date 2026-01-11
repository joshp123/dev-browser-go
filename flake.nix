{
  description = "dev-browser-go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = "0.1.0";

        devBrowserGo = pkgs.buildGoModule {
          pname = "dev-browser-go";
          inherit version;
          src = self;
          subPackages = [ "cmd/dev-browser-go" ];
          vendorHash = "sha256-pggjWBdRjfZ5Qs/i3bubhyRAvXFqYg2BAiQ3j7ov9tU=";
          ldflags = [ "-s" "-w" ];
          nativeBuildInputs = [ pkgs.makeWrapper ];
          postInstall = ''
            wrapProgram $out/bin/dev-browser-go \
              --set PLAYWRIGHT_BROWSERS_PATH ${pkgs.playwright-driver.browsers}
          '';
          meta = with pkgs.lib; {
            description = "Ref-based browser automation CLI+daemon (Go, Playwright)";
            homepage = "https://github.com/joshp123/dev-browser-go";
            license = licenses.agpl3Plus;
            platforms = platforms.unix;
            mainProgram = "dev-browser-go";
          };
        };
      in {
        packages = {
          dev-browser-go = devBrowserGo;
          default = devBrowserGo;
        };

        apps = {
          dev-browser-go = { type = "app"; program = "${devBrowserGo}/bin/dev-browser-go"; };
          default = { type = "app"; program = "${devBrowserGo}/bin/dev-browser-go"; };
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.playwright-driver ];
        };
      });
}
