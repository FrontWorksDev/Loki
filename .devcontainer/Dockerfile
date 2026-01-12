FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

RUN apt-get update && apt-get install -y curl tar \
    && ARCH=$(uname -m) \
    && if [ "$ARCH" = "aarch64" ]; then \
         URL="https://github.com/neovim/neovim/releases/latest/download/nvim-linux-arm64.tar.gz"; \
       else \
         URL="https://github.com/neovim/neovim/releases/latest/download/nvim-linux64.tar.gz"; \
       fi \
    && curl -LO "$URL" \
    && tar -C /usr/local -xzf $(basename "$URL") --strip-components=1 \
    && rm $(basename "$URL")

RUN mkdir -p /home/vscode/.local/state /home/vscode/.local/share /home/vscode/.config \
    && chown -R vscode:vscode /home/vscode

USER vscode
RUN go install golang.org/x/tools/gopls@v0.20.0 \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/air-verse/air@v1.62.0 \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
