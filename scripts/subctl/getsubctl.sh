#!/bin/bash
set -e

get_operating_system() {
  case $(uname -s) in

    Darwin*) echo "darwin" ;;
    Linux*) echo "linux" ;;
    CYGWIN*) echo "windows" ;;
    MINGW*) echo "windows" ;;
    *)
    error "This installer only works on Linux, macos and Windows. Found $(uname -s)"
    return 1;;
  esac
}

get_architecture() {
  case $(uname -m) in

    x86_64) echo "amd64" ;;
    amd64) echo "amd64" ;;
    i?86) echo "386" ;;
    *)
    error "This installer only supports x86_64 and i386 architectures. Found $(uname -m)"
    return 1;;
  esac
}

command_exists() {
  command -v "$@" > /dev/null 2>&1
}

download_command() {
  if command_exists curl; then
    echo "curl -fsSL"
  elif command_exists wget; then
    echo "wget -qO-"
  else
    error "curl and wget are missing"
    return 1
  fi
}

get=$(download_command)

get_subctl_release_url() {
  local draft_filter="cat"
  local url
  case ${VERSION} in
    rc) url=https://api.github.com/repos/submariner-io/submariner-operator/releases
           draft_filter="grep \-rc"
           ;;
    latest) url=https://api.github.com/repos/submariner-io/submariner-operator/releases/latest ;;
    devel) url=https://api.github.com/repos/submariner-io/submariner-operator/releases/tags/devel ;;
    *) url=https://api.github.com/repos/submariner-io/submariner-operator/releases/tags/${VERSION} ;;
  esac

  ${get} "${url}" | grep "browser_download_url.*-${os}-${architecture}" | ${draft_filter} | head -n 1 | cut -d\" -f 4
}

finish_cleanup() {
  [ -z "${tmpdir}" ] || rm -rf "$tmpdir"
}

get_subctl() {
  tmpdir=$(mktemp -d)

  cd "${tmpdir}"

  case ${url} in
    *tar.xz)
      ${get} "${url}" | tar xfJ -
      # shellcheck disable=SC2086
      install_subctl subctl*/subctl*${os}-${architecture}*
      ;;
    *) # non tar.xz releases (older)
      filename=$(basename "${url}")
      ${get} "${url}" > "${filename}"
      install_subctl "${filename}"
      ;;
  esac
}

install_subctl() {
  local bin=$1
  local bin_file
  local destdir=~/.local/bin
  local dest=${destdir}/subctl

  install -D "${bin}" "${dest}"

  bin_file=$(basename "${bin}")
  echo "${bin_file} has been installed as ${dest}"
  printf "This provides "
  ${dest} version
  if [ "$(command -v subctl)" != "${dest}" ]; then
    echo ""
    echo "please update your path (and consider adding it to ~/.profile):
    export PATH=\$PATH:${destdir}
    "
  fi
}

VERSION="${VERSION:-latest}"
os=$(get_operating_system)
architecture=$(get_architecture)
url=$(get_subctl_release_url)

echo "Installing subctl version $VERSION"
echo "  OS detected:           ${os}"
echo "  Architecture detected: ${architecture}"
echo "  Download URL:          ${url}"
echo ""
echo "Downloading..."

trap finish_cleanup EXIT

get_subctl
