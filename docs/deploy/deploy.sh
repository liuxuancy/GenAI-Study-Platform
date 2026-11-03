#!/usr/bin/env bash
# 更新代码、构建前后端、重启 systemd 服务。需已配置 backend/.env 与 caveclass.service。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck disable=SC1091
[ -f "$SCRIPT_DIR/env" ] && source "$SCRIPT_DIR/env"

REPO_DIR="${REPO_DIR:-$ROOT}"
SERVICE_NAME="${SERVICE_NAME:-caveclass}"
DEFAULT_BRANCH="$(git -C "$REPO_DIR" remote show origin 2>/dev/null | awk '/HEAD branch/ {print $NF}')"
BRANCH="${BRANCH:-${DEFAULT_BRANCH:-master}}"

cd "$REPO_DIR"
git fetch origin
if git show-ref --verify --quiet "refs/heads/$BRANCH"; then
  git checkout "$BRANCH"
elif git show-ref --verify --quiet "refs/remotes/origin/$BRANCH"; then
  git checkout -B "$BRANCH" "origin/$BRANCH"
else
  echo "error: branch $BRANCH not found; set BRANCH in docs/deploy/env (git branch -a)"
  exit 1
fi
git pull --ff-only origin "$BRANCH"

# 可选：INIT_DB=1 bash deploy.sh 会重建 SQLite（停服务后再用）
if [ "${INIT_DB:-0}" = "1" ]; then
  systemctl stop "$SERVICE_NAME" 2>/dev/null || true
  bash "$SCRIPT_DIR/init-db.sh"
fi

# 前端静态资源 -> frontend/dist
cd "$REPO_DIR/frontend"
corepack enable pnpm 2>/dev/null || true
pnpm install --frozen-lockfile
pnpm build

# 后端二进制 -> backend/bin/caveclass-server
cd "$REPO_DIR/backend"
go build -trimpath -ldflags="-s -w" -o bin/caveclass-server ./cmd/server

systemctl restart "$SERVICE_NAME"
systemctl is-active --quiet "$SERVICE_NAME"
echo "OK: deployed $(git rev-parse --short HEAD)"
