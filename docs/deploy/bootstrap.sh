#!/usr/bin/env bash
# 首次：克隆仓库、准备 .env、建库、编译。宝塔里先装好 Git / Go 1.26+ / Node 20+ / pnpm。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
[ -f "$SCRIPT_DIR/env" ] && source "$SCRIPT_DIR/env"

REPO_DIR="${REPO_DIR:?设置 REPO_DIR，或复制 env.example 为 env 并 source}"
GIT_REPO="${GIT_REPO:?设置 GIT_REPO}"
DEFAULT_BRANCH="$(git -C "$REPO_DIR" remote show origin 2>/dev/null | awk '/HEAD branch/ {print $NF}')"
BRANCH="${BRANCH:-${DEFAULT_BRANCH:-master}}"

if [ ! -d "$REPO_DIR/.git" ]; then
  git clone -b "$BRANCH" "$GIT_REPO" "$REPO_DIR"
fi

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

# 首次复制 backend/.env，在面板或 SSH 里填 JWT_SECRET、COS、AI 等后再部署
if [ ! -f backend/.env ]; then
  cp backend/.env.example backend/.env
  echo "已生成 backend/.env，请编辑后重新运行 deploy.sh"
  exit 0
fi

export REPO_DIR
DB_FILE="$REPO_DIR/backend/${DB_PATH:-data/caveclass.db}"
if [ ! -f "$DB_FILE" ] || [ "${INIT_DB:-0}" = "1" ]; then
  bash "$SCRIPT_DIR/init-db.sh"
fi
bash "$SCRIPT_DIR/deploy.sh"
