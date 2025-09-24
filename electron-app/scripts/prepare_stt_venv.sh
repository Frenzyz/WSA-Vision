#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PY_DIR="$ROOT_DIR/python"

mkdir -p "$PY_DIR"

if [ ! -d "$PY_DIR/venv" ]; then
  python3 -m venv "$PY_DIR/venv"
fi

source "$PY_DIR/venv/bin/activate"
python -m pip install --upgrade pip
python -m pip install whisperx

# Freeze dependencies for reference
python -m pip freeze > "$PY_DIR/requirements.txt"

echo "STT venv prepared at $PY_DIR/venv"

