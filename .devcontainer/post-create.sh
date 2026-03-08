direnv allow
devbox install
devbox run go install -v golang.org/x/tools/gopls@latest
direnv reload