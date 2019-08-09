build: find . -name '*.go' | entr -r vgo build -o nix-private-cached
server: echo 'nix-private-cached' | entr -r ./nix-private-cached
