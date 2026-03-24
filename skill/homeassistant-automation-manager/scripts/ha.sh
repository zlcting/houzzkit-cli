#!/usr/bin/env bash
# Home Assistant CLI wrapper
# Usage: ha.sh <command> [args...]

set -euo pipefail

CONFIG_FILE="${HA_CONFIG:-$HOME/.config/home-assistant/config.json}"

# Load config
if [[ -f "$CONFIG_FILE" ]]; then
  HA_URL="${HA_URL:-$(jq -r '.url // empty' "$CONFIG_FILE")}"
  HA_TOKEN="${HA_TOKEN:-$(jq -r '.token // empty' "$CONFIG_FILE")}"
fi

: "${HA_URL:?Set HA_URL or configure $CONFIG_FILE}"
: "${HA_TOKEN:?Set HA_TOKEN or configure $CONFIG_FILE}"

cmd="${1:-help}"
shift || true

api() {
  curl -s -H "Authorization: Bearer $HA_TOKEN" -H "Content-Type: application/json" "$@"
}

case "$cmd" in
  get-automation-list)
    # Get automation list: ha.sh get-automation-list
    api "$HA_URL/api/states" | jq '.[] | select(.entity_id | startswith("automation.")) | {entity_id: .entity_id, state: .state, friendly_name: .attributes.friendly_name, id: .attributes.id, last_triggered: .attributes.last_triggered}'
    ;;

  get-run-action-entity)
    # Get run action entity 获取可执行的动作的实体 : ha.sh get-run-action-list
    api "$HA_URL/api/states" | jq '[.[] | select(.entity_id | test("^text\\.houzzkit_(?!.*_xun_wen_hou_).*_zhi_xing_ming_ling$")) | {entity_id: .entity_id, state: .state, friendly_name: .attributes.friendly_name, id: .attributes.id, last_triggered: .attributes.last_triggered}]'    
    ;;

  create-automation)
    # Create automation: ha.sh create-automation '{"description":"","mode":"single","triggers":[{"trigger":"state","entity_id":["<entity_id>"]}],"conditions":[],"actions":[{"action":"text.set_value","metadata":{},"data":{"value":"开灯"},"target":{"entity_id":"text.houzzkit_edec_zhi_xing_ming_ling"}}],"alias":"开灯"}'
    data="${1:?Usage: ha.sh create-automation <json_data>}"
    timestamp=$(date +%s)
    api -X POST "$HA_URL/api/config/automation/config/$timestamp" -d "$data"
    echo "✓ Automation created"
    ;;  
  info)
    # Get HA instance info
    api "$HA_URL/api/" | jq
    ;;

  help|*)
    cat <<EOF
Home Assistant CLI

Usage: ha.sh <command> [args...]

Commands:
  state <entity>              Get entity state
  states <entity>             Get full entity state with attributes
  on <entity> [brightness]    Turn on (optional brightness 0-255)
  off <entity>                Turn off
  toggle <entity>             Toggle on/off
  scene <name>                Activate scene
  script <name>               Run script
  automation <name>           Trigger automation
  climate <entity> <temp>     Set temperature
  list [domain]               List entities (lights, switches, all)
  search <pattern>            Search entities by name
  call <domain> <svc> [json]  Call any service
  info                        Get HA instance info

Environment:
  HA_URL    Home Assistant URL (required)
  HA_TOKEN  Long-lived access token (required)

Examples:
  ha.sh on light.living_room 200
  ha.sh scene movie_night
  ha.sh list lights
  ha.sh search kitchen
EOF
    ;;
esac
