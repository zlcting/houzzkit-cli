package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Entity struct {
	EntityID   string                 `json:"entity_id"`
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

var generateAutomationsCmd = &cobra.Command{
	Use:   "generate-automations",
	Short: "Generate automations.yaml based on HA scenes and virtual events",
	Long:  `Fetch entities from HA /api/states, extract scenes and virtual event entities, and generate automations.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

			id := fmt.Sprintf("%d", time.Now().UnixMilli())
			alias := strings.ReplaceAll(strings.ReplaceAll(friendlyName+"模式", "_", ""), " ", "")
			entityID := scene.EntityID

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
`, id, alias, eventID, alias, entityID)

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
