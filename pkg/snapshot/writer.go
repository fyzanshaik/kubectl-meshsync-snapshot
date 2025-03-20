package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)
func SaveToFile(resources []models.KubernetesResource, filePath string) error {
	snapshot := map[string]interface{}{
		"version":     "v1",
		"timestamp":   time.Now().Format(time.RFC3339),
		"resources":   resources,
		"cluster_id":  getClusterID(resources),
		"plugin_info": getPluginInfo(),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot to JSON: %w", err)
	}
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write snapshot to file: %w", err)
	}
	return nil
}
func getClusterID(resources []models.KubernetesResource) string {
	if len(resources) > 0 {
		return resources[0].ClusterID
	}
	return "unknown"
}
func getPluginInfo() map[string]string {
	return map[string]string{
		"name":        "kubectl-meshsync_snapshot",
		"version":     "0.1.0",
		"description": "A kubectl plugin for capturing Kubernetes cluster state using MeshSync",
		"created_at":  time.Now().Format(time.RFC3339),
	}
}