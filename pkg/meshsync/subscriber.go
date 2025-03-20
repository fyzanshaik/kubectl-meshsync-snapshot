package meshsync

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
	"github.com/nats-io/nats.go"
)
func CollectResources(natsURL string, duration time.Duration) ([]models.KubernetesResource, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()
	var resources []models.KubernetesResource
	resourceChan := make(chan models.KubernetesResource, 1000)
	doneChan := make(chan bool)
	sub, err := nc.Subscribe("meshery.meshsync.core", func(msg *nats.Msg) {
		var message struct {
			Object models.KubernetesResource `json:"Object"`
		}
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			fmt.Printf("Error unmarshaling message: %v\n", err)
			return
		}
		resourceChan <- message.Object
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to NATS topic: %w", err)
	}
	defer sub.Unsubscribe()
	go func() {
		time.Sleep(duration)
		doneChan <- true
	}()
	for {
		select {
		case resource := <-resourceChan:
			resources = append(resources, resource)
		case <-doneChan:
			return resources, nil
		}
	}
}