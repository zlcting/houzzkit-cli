package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type HAConfig struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func loadHAConfig() (string, string, error) {
	haURL := strings.TrimSpace(os.Getenv("HA_URL"))
	haToken := strings.TrimSpace(os.Getenv("HA_TOKEN"))

	configPath := os.Getenv("HA_CONFIG")
	if configPath == "" {
		home := os.Getenv("HOME")
		configPath = filepath.Join(home, ".config", "home-assistant", "config.json")
	}

	if haURL == "" || haToken == "" {
		if stat, err := os.Stat(configPath); err == nil && !stat.IsDir() {
			data, err := os.ReadFile(configPath)
			if err != nil {
				return "", "", fmt.Errorf("read config file failed: %w", err)
			}

			var c HAConfig
			if err := json.Unmarshal(data, &c); err != nil {
				return "", "", fmt.Errorf("parse config file failed: %w", err)
			}

			if haURL == "" {
				haURL = strings.TrimSpace(c.URL)
			}
			if haToken == "" {
				haToken = strings.TrimSpace(c.Token)
			}
		}
	}

	if haURL == "" {
		return "", "", fmt.Errorf("HA_URL not set and not found in %s", configPath)
	}
	if haToken == "" {
		return "", "", fmt.Errorf("HA_TOKEN not set and not found in %s", configPath)
	}

	return haURL, haToken, nil
}

var haEntityDomains = map[string]bool{
	"light": true, "switch": true, "climate": true, "cover": true,
	"fan": true, "lock": true, "vacuum": true, "media_player": true,
	"remote": true, "sensor": true, "binary_sensor": true,
}

func fetchHAStates(haURL, haToken string) ([]map[string]any, error) {
	apiURL := strings.TrimRight(haURL, "/") + "/api/states"
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+haToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HA API returned %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var list []map[string]any
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	return list, nil
}

func filterDeviceEntities(states []map[string]any) []map[string]any {
	var results []map[string]any
	for _, item := range states {
		entityID, _ := item["entity_id"].(string)
		if entityID == "" {
			continue
		}
		parts := strings.Split(entityID, ".")
		if len(parts) < 2 {
			continue
		}
		domain := parts[0]
		if !haEntityDomains[domain] {
			continue
		}

		state, _ := item["state"].(string)
		if state == "" || state == "unavailable" || state == "unknown" {
			continue
		}

		attrs, _ := item["attributes"].(map[string]any)
		friendlyName, _ := attrs["friendly_name"].(string)

		results = append(results, map[string]any{
			"entity_id":     entityID,
			"domain":        domain,
			"state":         state,
			"friendly_name": friendlyName,
			"attributes":    attrs,
		})
	}
	return results
}

func findEntityMatches(query string, entities []map[string]any) []map[string]any {
	var matches []map[string]any
	lower := strings.ToLower(query)
	for _, e := range entities {
		entityID, _ := e["entity_id"].(string)
		friendlyName, _ := e["friendly_name"].(string)
		if strings.Contains(strings.ToLower(entityID), lower) || strings.Contains(strings.ToLower(friendlyName), lower) {
			matches = append(matches, e)
		}
	}
	return matches
}

func promptSelectEntity(candidates []map[string]any, prompt string) (map[string]any, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates")
	}
	fmt.Println(prompt)
	for i, e := range candidates {
		fmt.Printf("%d) %s (%s)\n", i+1, e["entity_id"], e["friendly_name"])
	}

	var idx int
	_, err := fmt.Scanln(&idx)
	if err != nil {
		return nil, err
	}
	if idx <= 0 || idx > len(candidates) {
		return nil, fmt.Errorf("invalid selection")
	}
	return candidates[idx-1], nil
}

func buildAutomationName(triggerEntity, targetEntity, actionOrder string) string {
	return fmt.Sprintf("%s -> %s [%s]", triggerEntity, targetEntity, actionOrder)
}

func postCreateAutomation(haURL, haToken, triggerEntityID, targetEntityID, actionOrder, alias string) error {
	payload := map[string]any{
		"description": "",
		"mode":        "single",
		"triggers": []any{
			map[string]any{
				"trigger":   "state",
				"entity_id": []string{triggerEntityID},
			},
		},
		"conditions": []any{},
		"actions": []any{
			map[string]any{
				"action":   "text.set_value",
				"metadata": map[string]any{},
				"data": map[string]any{
					"value": actionOrder,
				},
				"target": map[string]any{
					"entity_id": targetEntityID,
				},
			},
		},
		"alias": alias,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	apiURL := strings.TrimRight(haURL, "/") + "/api/config/automation/config/" + timestamp

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+haToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HA API returned %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Created automation: %s\n", string(body))
	return nil
}

func fetchRunActionEntities(haURL, haToken string) ([]map[string]any, error) {
	states, err := fetchHAStates(haURL, haToken)
	if err != nil {
		return nil, err
	}
	var results []map[string]any
	for _, item := range states {
		entityID, _ := item["entity_id"].(string)
		if !strings.HasPrefix(entityID, "text.houzzkit_") {
			continue
		}
		if !strings.HasSuffix(entityID, "_zhi_xing_ming_ling") {
			continue
		}
		if strings.Contains(entityID, "_xun_wen_hou_") {
			continue
		}

		state, _ := item["state"].(string)
		attrs, _ := item["attributes"].(map[string]any)
		friendlyName, _ := attrs["friendly_name"].(string)

		results = append(results, map[string]any{
			"entity_id":      entityID,
			"state":          state,
			"friendly_name":  friendlyName,
			"attributes":     attrs,
		})
	}
	return results, nil
}

func findRunActionMatches(query string, actions []map[string]any) []map[string]any {
	var matches []map[string]any
	q := strings.ToLower(query)
	for _, a := range actions {
		entityID, _ := a["entity_id"].(string)
		friendlyName, _ := a["friendly_name"].(string)
		if strings.Contains(strings.ToLower(entityID), q) || strings.Contains(strings.ToLower(friendlyName), q) {
			matches = append(matches, a)
		}
	}
	return matches
}

var haCmd = &cobra.Command{
	Use:   "ha",
	Short: "HA-related utilities",
	Long:  `Home Assistant utility commands that share HA_URL/HA_TOKEN configuration.`,
}

// ./houzzkit_tool ha get-automation-list
var haAutomationCmd = &cobra.Command{
	Use:   "get-automation-list",
	Short: "List Home Assistant automations",
	Run: func(cmd *cobra.Command, args []string) {
		haURL, haToken, err := loadHAConfig()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		apiURL := strings.TrimRight(haURL, "/") + "/api/states"
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			fmt.Printf("create request failed: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+haToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("HA API returned %d: %s\n", resp.StatusCode, string(body))
			os.Exit(1)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("read response failed: %v\n", err)
			os.Exit(1)
		}

		var list []map[string]any
		if err := json.Unmarshal(data, &list); err != nil {
			fmt.Printf("parse JSON failed: %v\n", err)
			os.Exit(1)
		}

		var results []map[string]any
		for _, item := range list {
			entityID, _ := item["entity_id"].(string)
			if strings.HasPrefix(entityID, "automation.") {
				state, _ := item["state"].(string)
				attrs, _ := item["attributes"].(map[string]any)
				friendlyName, _ := attrs["friendly_name"].(string)
				id, _ := attrs["id"].(string)
				lastTriggered := ""
				if lt, ok := attrs["last_triggered"].(string); ok {
					lastTriggered = lt
				}
				results = append(results, map[string]any{
					"entity_id":      entityID,
					"state":          state,
					"friendly_name":  friendlyName,
					"id":             id,
					"last_triggered": lastTriggered,
				})
			}
		}

		enc, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			fmt.Printf("encode JSON failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(enc))
	},
}

var haRunActionCmd = &cobra.Command{
	Use:   "get-run-action-list",
	Short: "List runtask entities for houzzkit actions",
	Run: func(cmd *cobra.Command, args []string) {
		haURL, haToken, err := loadHAConfig()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		apiURL := strings.TrimRight(haURL, "/") + "/api/states"
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			fmt.Printf("create request failed: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+haToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("HA API returned %d: %s\n", resp.StatusCode, string(body))
			os.Exit(1)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("read response failed: %v\n", err)
			os.Exit(1)
		}

		var list []map[string]any
		if err := json.Unmarshal(data, &list); err != nil {
			fmt.Printf("parse JSON failed: %v\n", err)
			os.Exit(1)
		}

		var results []map[string]any
		for _, item := range list {
			entityID, _ := item["entity_id"].(string)
			if !strings.HasPrefix(entityID, "text.houzzkit_") {
				continue
			}
			if !strings.HasSuffix(entityID, "_zhi_xing_ming_ling") {
				continue
			}
			if strings.Contains(entityID, "_xun_wen_hou_") {
				continue
			}

			state, _ := item["state"].(string)
			attrs, _ := item["attributes"].(map[string]any)
			friendlyName, _ := attrs["friendly_name"].(string)
			id, _ := attrs["id"].(string)
			lastTriggered := ""
			if lt, ok := attrs["last_triggered"].(string); ok {
				lastTriggered = lt
			}

			results = append(results, map[string]any{
				"entity_id":      entityID,
				"state":          state,
				"friendly_name":  friendlyName,
				"id":             id,
				"last_triggered": lastTriggered,
			})
		}

		enc, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			fmt.Printf("encode JSON failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(enc))
	},
}

var haCreateAutomationCmd = &cobra.Command{
	Use:   "create-automation <trigger_entity_id> <target_entity_id> <action_order> <alias>",
	Short: "Create automation config in HA with highways simplified fields",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		haURL, haToken, err := loadHAConfig()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		triggerEntityID := args[0]
		targetEntityID := args[1]
		actionOrder := args[2]
		alias := args[3]

		payload := map[string]any{
			"description": "",
			"mode":        "single",
			"triggers": []any{
				map[string]any{
					"trigger":   "state",
					"entity_id": []string{triggerEntityID},
				},
			},
			"conditions": []any{},
			"actions": []any{
				map[string]any{
					"action":   "text.set_value",
					"metadata": map[string]any{},
					"data": map[string]any{
						"value": actionOrder,
					},
					"target": map[string]any{
						"entity_id": targetEntityID,
					},
				},
			},
			"alias": alias,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("json marshal failed: %v\n", err)
			os.Exit(1)
		}

		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		apiURL := strings.TrimRight(haURL, "/") + "/api/config/automation/config/" + timestamp

		req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(string(jsonData)))
		if err != nil {
			fmt.Printf("create request failed: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+haToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			fmt.Printf("HA API returned %d: %s\n", resp.StatusCode, string(body))
			os.Exit(1)
		}

		fmt.Printf("Created automation: %s\n", string(body))
	},
}

var haGetEntityCmd = &cobra.Command{
	Use:   "get-entity-list",
	Short: "List entities (light, switch, climate, etc.) with available state",
	Run: func(cmd *cobra.Command, args []string) {
		haURL, haToken, err := loadHAConfig()
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}

		apiURL := strings.TrimRight(haURL, "/") + "/api/states"
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			fmt.Printf("create request failed: %v\n", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+haToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("request failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("HA API returned %d: %s\n", resp.StatusCode, string(body))
			os.Exit(1)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("read response failed: %v\n", err)
			os.Exit(1)
		}

		var list []map[string]any
		if err := json.Unmarshal(data, &list); err != nil {
			fmt.Printf("parse JSON failed: %v\n", err)
			os.Exit(1)
		}

		// Device types to include
		deviceTypes := map[string]bool{
			"light": true, "switch": true, "climate": true, "cover": true,
			"fan": true, "lock": true, "vacuum": true, "media_player": true,
			"remote": true, "sensor": true, "binary_sensor": true,
		}

		var results []map[string]any
		for _, item := range list {
			entityID, _ := item["entity_id"].(string)
			if entityID == "" {
				continue
			}

			// Extract domain from entity_id (e.g., "light" from "light.living_room")
			parts := strings.Split(entityID, ".")
			if len(parts) < 2 {
				continue
			}
			domain := parts[0]

			// Filter by device type
			if !deviceTypes[domain] {
				continue
			}

			state, _ := item["state"].(string)
			// Skip if state is empty or unavailable
			if state == "" || state == "unavailable" || state == "unknown" {
				continue
			}

			attrs, _ := item["attributes"].(map[string]any)
			friendlyName, _ := attrs["friendly_name"].(string)

			results = append(results, map[string]any{
				"entity_id":     entityID,
				"domain":        domain,
				"state":         state,
				"friendly_name": friendlyName,
				"attributes":    attrs,
			})
		}

		enc, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			fmt.Printf("encode JSON failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(enc))
	},
}

func init() {
	rootCmd.AddCommand(haCmd)
	haCmd.AddCommand(haAutomationCmd)
	haCmd.AddCommand(haRunActionCmd)
	haCmd.AddCommand(haCreateAutomationCmd)
	haCmd.AddCommand(haGetEntityCmd)
}
