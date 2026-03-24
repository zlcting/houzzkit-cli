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

func init() {
	rootCmd.AddCommand(haCmd)
	haCmd.AddCommand(haAutomationCmd)
	haCmd.AddCommand(haRunActionCmd)
	haCmd.AddCommand(haCreateAutomationCmd)
}
