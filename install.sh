#!/usr/bin/env bash
# install.sh — instala o CLI `tmjlite` (macOS Apple Silicon e Linux x64).
#
#   curl -fsSL https://tmjlite.com.br/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/TMJacometti/tmjlite/main/install.sh | bash
#
# O que faz: descobre o sistema, baixa tmjlite-cli-<plataforma>-<versão>.tar.gz
# da release pública no GitHub, confere o SHA-256 contra o SHA256SUMS da mesma
# release (sem conferência não instala) e copia o binário para /usr/local/bin
# se der para escrever lá, senão para ~/.local/bin. Não usa sudo.
#
# Variáveis opcionais:
#   TMJLITE_VERSION      versão a instalar (ex.: 0.5.2 ou v0.5.2); padrão: a última
#   TMJLITE_INSTALL_DIR  diretório de destino (padrão: /usr/local/bin ou ~/.local/bin)
#   TMJLITE_REPO         repositório das releases (padrão: TMJacometti/tmjlite)
#   TMJLITE_BASE_URL     URL base dos arquivos (padrão: a release no GitHub; para espelhos/testes)
#
# Windows: baixe o .zip em https://tmjlite.com.br/#cli.
set -euo pipefail

# Tudo dentro de main(), chamada na última linha: se o download deste script
# for cortado no meio, o bash não executa um pedaço dele.
main() {
    local repo="${TMJLITE_REPO:-TMJacometti/tmjlite}"
    local version="${TMJLITE_VERSION:-}"
    local tmp=""

    say() { printf '%s\n' "$*"; }
    err() { printf 'erro: %s\n' "$*" >&2; }
    die() { err "$*"; exit 1; }
    have() { command -v "$1" >/dev/null 2>&1; }

    # fetch <url> <arquivo>  — falha (sem gravar página de erro) em 4xx/5xx
    fetch() {
        if have curl; then
            curl -fsSL --proto '=https,file' --retry 3 -o "$2" "$1"
        elif have wget; then
            wget -q -O "$2" "$1"
        else
            die "preciso de curl ou wget para baixar"
        fi
    }

    sha256_of() {
        if have sha256sum; then
            sha256sum "$1" | awk '{print $1}'
        elif have shasum; then
            shasum -a 256 "$1" | awk '{print $1}'
        elif have openssl; then
            openssl dgst -sha256 "$1" | awk '{print $NF}'
        else
            return 1
        fi
    }

    # --- plataforma: só as que a release publica -----------------------------
    local os arch platform
    os="$(uname -s)"
    arch="$(uname -m)"
    case "$os" in
        Darwin)
            # Terminal rodando sob Rosetta num Mac Apple Silicon diz x86_64.
            if [ "$arch" = "x86_64" ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = "1" ]; then
                arch="arm64"
            fi
            case "$arch" in
                arm64|aarch64) platform="macos-arm64" ;;
                *) die "macOS Intel ($arch) não tem binário publicado — só Apple Silicon (arm64)." ;;
            esac
            ;;
        Linux)
            case "$arch" in
                x86_64|amd64) platform="linux-x64" ;;
                *) die "Linux $arch não tem binário publicado — só x86_64." ;;
            esac
            local libc
            libc="$(ldd --version 2>&1 || true)"
            case "$libc" in
                *musl*) die "este Linux usa musl (Alpine?) e o binário publicado é glibc." ;;
            esac
            ;;
        *)
            die "sistema não suportado por este instalador: $os. No Windows, baixe o .zip em https://tmjlite.com.br/#cli"
            ;;
    esac

    have tar || die "preciso do comando tar"

    tmp="$(mktemp -d 2>/dev/null || mktemp -d -t tmjlite)"
    # shellcheck disable=SC2064
    trap "rm -rf '$tmp'" EXIT

    # --- versão ---------------------------------------------------------------
    version="${version#v}"
    if [ -z "$version" ]; then
        # O repo público também tem releases do Studio (studio-v*), então
        # "releases/latest" não serve: a versão do CLI vem do manifest.json que
        # o CI publica na raiz do repo (primeira chave de topo, indentação 2).
        if fetch "https://raw.githubusercontent.com/${repo}/main/manifest.json" "$tmp/manifest.json" 2>/dev/null; then
            version="$(sed -n 's/^  "version": *"\([^"]*\)".*/\1/p' "$tmp/manifest.json" | sed -n 1p)"
        fi
        if [ -z "$version" ]; then
            # Plano B: lista de releases da API, primeira tag vX.Y.Z.
            if fetch "https://api.github.com/repos/${repo}/releases?per_page=30" "$tmp/releases.json" 2>/dev/null; then
                version="$(sed -n 's/.*"tag_name": *"v\([0-9][^"]*\)".*/\1/p' "$tmp/releases.json" | sed -n 1p)"
            fi
        fi
        [ -n "$version" ] || die "não consegui descobrir a última versão; informe TMJLITE_VERSION=X.Y.Z"
    fi
    case "$version" in
        *[!0-9A-Za-z.+-]*|"") die "versão inválida: '$version'" ;;
    esac

    local base="${TMJLITE_BASE_URL:-https://github.com/${repo}/releases/download/v${version}}"
    local dir_name="tmjlite-cli-${platform}"
    local tarball="${dir_name}-${version}.tar.gz"

    say "TMJLite ${version} (${platform})"
    say "  baixando ${base}/${tarball}"
    fetch "${base}/${tarball}" "$tmp/${tarball}" \
        || die "não achei ${tarball} na release v${version} (o instalador vale para versões que publicam .tar.gz; confira https://github.com/${repo}/releases)"
    fetch "${base}/SHA256SUMS" "$tmp/SHA256SUMS" \
        || die "a release v${version} não tem SHA256SUMS — sem conferência eu não instalo"

    # --- SHA-256 (fecha em qualquer dúvida) -------------------------------------
    local want got
    want="$(awk -v f="$tarball" '$2 == f || $2 == "*" f { print $1; exit }' "$tmp/SHA256SUMS")"
    [ -n "$want" ] || die "SHA256SUMS não lista ${tarball} — não instalo"
    got="$(sha256_of "$tmp/${tarball}")" || die "preciso de sha256sum, shasum ou openssl para conferir o download"
    if [ "$got" != "$want" ]; then
        err "SHA-256 não confere para ${tarball}"
        err "  esperado: $want"
        err "  obtido:   $got"
        exit 1
    fi
    say "  SHA-256 conferido"

    # --- extrai só o binário ------------------------------------------------------
    mkdir "$tmp/x"
    tar -xzf "$tmp/${tarball}" -C "$tmp/x" "${dir_name}/tmjlite" \
        || die "o pacote não tem ${dir_name}/tmjlite"
    local bin="$tmp/x/${dir_name}/tmjlite"
    [ -f "$bin" ] || die "binário ausente depois de extrair"
    chmod 755 "$bin"

    # --- destino (nunca sudo) --------------------------------------------------------
    local dest="${TMJLITE_INSTALL_DIR:-}"
    if [ -z "$dest" ]; then
        if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
            dest="/usr/local/bin"
        else
            dest="${HOME:?HOME não definido}/.local/bin"
        fi
    fi
    mkdir -p "$dest" 2>/dev/null || die "não consegui criar $dest — escolha outro com TMJLITE_INSTALL_DIR=/caminho"
    [ -w "$dest" ] || die "sem permissão de escrita em $dest — escolha outro com TMJLITE_INSTALL_DIR=/caminho (este script não usa sudo)"

    # copia com nome temporário e renomeia: troca atômica, mesmo com o tmjlite antigo em uso
    cp "$bin" "$dest/.tmjlite.new.$$"
    chmod 755 "$dest/.tmjlite.new.$$"
    mv -f "$dest/.tmjlite.new.$$" "$dest/tmjlite"

    # Teste de fumaça: o binário roda nesta máquina?
    if ! "$dest/tmjlite" --version >/dev/null 2>&1; then
        err "instalei em $dest/tmjlite, mas '$dest/tmjlite --version' falhou nesta máquina."
        if [ "$os" = "Linux" ]; then
            err "Causa comum: glibc mais antiga que a do build. Confira com: ldd $dest/tmjlite"
        else
            err "Veja a solução de problemas em https://tmjlite.com.br/manual.html"
        fi
        exit 1
    fi

    say "  instalado: $dest/tmjlite ($("$dest/tmjlite" --version 2>/dev/null | sed -n 1p))"
    case ":${PATH}:" in
        *":${dest}:"*)
            say ""
            say "Pronto. Experimente:  tmjlite minha-base.tmjp"
            ;;
        *)
            say ""
            say "$dest não está no seu PATH. Adicione ao arquivo de perfil do seu shell (~/.zshrc, ~/.bashrc…):"
            say "  export PATH=\"$dest:\$PATH\""
            say "Depois abra um terminal novo e rode:  tmjlite minha-base.tmjp"
            ;;
    esac
}

main "$@"
