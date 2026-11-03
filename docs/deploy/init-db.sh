#!/usr/bin/env bash
# 用 docs/sql 建库并导入英文 seed。执行前务必停止 caveclass 后端。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="${REPO_DIR:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
DB_REL="${DB_PATH:-data/caveclass.db}"
DB_FILE="$ROOT/backend/$DB_REL"

mkdir -p "$(dirname "$DB_FILE")"
rm -f "$DB_FILE" "${DB_FILE}-wal" "${DB_FILE}-shm"

sqlite3 "$DB_FILE" < "$ROOT/docs/sql/init.sql"
sqlite3 "$DB_FILE" < "$ROOT/docs/sql/seed.sql"

echo "OK: $DB_FILE"
