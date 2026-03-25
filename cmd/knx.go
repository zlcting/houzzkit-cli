package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Entity struct {
	EntityID   string                 `json:"entity_id"`
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

// 定义结构体来映射YAML配置
type Config struct {
	KNX struct {
		Light []struct {
			Name         string `yaml:"name"`
			Address      string `yaml:"address"`
			StateAddress string `yaml:"state_address"`
		} `yaml:"light"`
	} `yaml:"knx"`
}

var generateAutomationsCmd = &cobra.Command{
	Use:   "generate-automations",
	Short: "Generate automations.yaml based on HA scenes and virtual events",
	Long:  `Fetch entities from HA /api/states, extract scenes and virtual event entities, and generate automations.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id := time.Now().UnixMilli()
		haURL, haToken, err := loadHAConfig()
		if err != nil {
			return fmt.Errorf("failed to load HA config: %w", err)
		}

		// Fetch all states
		req, err := http.NewRequest("GET", haURL+"/api/states", nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+haToken)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to fetch states: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("HA API returned status %d", resp.StatusCode)
		}

		var entities []Entity
		if err := json.NewDecoder(resp.Body).Decode(&entities); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		// Filter scenes
		var scenes []Entity
		for _, e := range entities {
			if strings.HasPrefix(e.EntityID, "scene.") {
				scenes = append(scenes, e)
			}
		}

		// Find virtual event entity
		var eventID string
		for _, e := range entities {
			if eventTypes, ok := e.Attributes["event_types"].([]interface{}); ok {
				for _, et := range eventTypes {
					if etStr, ok := et.(string); ok && etStr == "Virtual Event Occurred" {
						eventID = e.EntityID
						break
					}
				}
				if eventID != "" {
					break
				}
			}
		}

		if eventID == "" {
			return fmt.Errorf("no virtual event entity found with 'Virtual Event Occurred'")
		}

		// Generate YAML
		var yamlBuilder strings.Builder
		for _, scene := range scenes {
			friendlyName, ok := scene.Attributes["friendly_name"].(string)
			if !ok {
				continue
			}

			alias := strings.ReplaceAll(strings.ReplaceAll(friendlyName+"模式", "_", ""), " ", "")
			entityID := scene.EntityID
			id = id + 1
			yaml := fmt.Sprintf(`- id: '%s'
  alias: %s
  description: ''
  triggers:
  - trigger: state
    entity_id:
    - %s
  conditions:
  - condition: template
    value_template: '{{ trigger.to_state.attributes[''Event Name''] == ''%s''}}'
  actions:
  - action: scene.turn_on
    metadata: {}
    data: {}
    target:
      entity_id: %s
`, fmt.Sprintf("%d", id), alias, eventID, alias, entityID)

			yamlBuilder.WriteString(yaml)
		}
		//增加双控自动化
		knxLightNames := []string{}
		knxCfgPath := "/var/hass/config/packages/houzzkit_knx.yaml"
		data, err := os.ReadFile(knxCfgPath)
		if err != nil {
			return fmt.Errorf("read knx config failed: %w", err)
		}
		var config Config
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			log.Fatalf("解析YAML失败: %v", err)
		}
		lights := config.KNX.Light
		for i, light := range lights {
			fmt.Printf("%2d. %-20s \n", i+1, light.Name)
			knxLightNames = append(knxLightNames, light.Name)
		}
		lightEntityMap := map[string]string{} // knx name -> light entity_id
		for _, e := range entities {
			if !strings.HasPrefix(e.EntityID, "light.") {
				continue
			}
			fn, ok := e.Attributes["friendly_name"].(string)
			if !ok {
				continue
			}
			for _, knxName := range knxLightNames {
				if fn == knxName {
					lightEntityMap[knxName] = e.EntityID
					break
				}
			}
		}

		// 3. 遍历 switch 实体找到 name 匹配的 switch entity_id
		switchMap := map[string]string{} // knx name -> switch entity list
		for _, e := range entities {
			if !strings.HasPrefix(e.EntityID, "switch.") {
				continue
			}
			fn, ok := e.Attributes["friendly_name"].(string)
			if !ok {
				continue
			}
			for knxName := range lightEntityMap {
				if fn == knxName || strings.Contains(fn, strings.ReplaceAll(knxName, "_", "")) {
					switchMap[knxName] = e.EntityID
				}
			}
		}
		if len(switchMap) == 0 {
			return fmt.Errorf("no matching switch entities found in HA states")
		}
		for knxName, lightEntity := range lightEntityMap {
			switchEntityID := switchMap[knxName]
			if len(switchEntityID) == 0 {
				continue
			}
			alias := knxName
			id = id + 1
			yaml := fmt.Sprintf(`- id: '%s'
  alias: %s
  description: ''
  triggers:
  - trigger: state
    entity_id:
    - %s
    - %s
  conditions: []
  actions:
  - delay: '00:00:00.5'
  - target:
      entity_id:
      - %s
      - %s
    action: homeassistant.turn_{{ trigger.to_state.state }}
  mode: single 
`, fmt.Sprintf("%d", id), alias, lightEntity, switchEntityID, lightEntity, switchEntityID)

			yamlBuilder.WriteString(yaml)
		}

		// Write to file
		err = os.WriteFile("automations.yaml", []byte(yamlBuilder.String()), 0644)
		if err != nil {
			return fmt.Errorf("failed to write automations.yaml: %w", err)
		}

		fmt.Println("automations.yaml generated successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateAutomationsCmd)
}
