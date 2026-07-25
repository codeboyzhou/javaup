#!/usr/bin/env bash

set -euo pipefail

image="javaup-demo"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "${script_dir}/../.." && pwd)"
skip_build=false

usage() {
  cat <<'EOF'
Usage: ./docs/demo/record.sh [--skip-build]

Options:
  --skip-build  Reuse the existing javaup-demo Docker image.
  -h, --help    Show this help message.
EOF
}

while (($# > 0)); do
  case "$1" in
    --skip-build)
      skip_build=true
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if ! docker version --format '{{.Server.Version}}' >/dev/null 2>&1; then
  echo "Docker is not available. Start Docker Desktop or the Docker daemon and try again." >&2
  exit 1
fi

cd "$repository_root"

if [[ "$skip_build" == false ]]; then
  echo "Building the javaup VHS recording image..."
  docker build --file docs/demo/Dockerfile --tag "$image" .
fi

echo "Recording docs/demo/demo.gif..."
docker run --rm --volume "${repository_root}:/jup" "$image" docs/demo/demo.tape

echo "Recorded ${repository_root}/docs/demo/demo.gif"
