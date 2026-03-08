#!/usr/bin/env bash
set -e

CONFIG_DIR="$HOME/.config/newsdigest"
mkdir -p "$CONFIG_DIR"

CONFIG_FILE="$CONFIG_DIR/config.json"

if [ ! -f "$CONFIG_FILE" ]; then
  read -rp "Enter your Anthropic API key: " API_KEY
  cat > "$CONFIG_FILE" <<EOF
{
  "anthropic_api_key": "$API_KEY",
  "last_fetch_date": "",
  "theme": "dark"
}
EOF
  chmod 600 "$CONFIG_FILE"
  echo "Config written to $CONFIG_FILE"
else
  echo "Config already exists at $CONFIG_FILE"
fi

echo "Setup complete. Run: ./herald"
