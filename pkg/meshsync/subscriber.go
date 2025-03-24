package meshsync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/utils"
	"github.com/nats-io/nats.go"
)
func CollectResources(ctx context.Context, natsURL string, options *models.Options) ([]*models.KubernetesResource, error) {
	if options.PreviewMode {
		return previewResources(options)
	}
	nc, err := nats.Connect(natsURL, 
		nats.ReconnectWait(300*time.Millisecond),  
		nats.MaxReconnects(5),                     
		nats.RetryOnFailedConnect(true),
		nats.Timeout(3*time.Second),               
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if options.VerboseMode {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()
	var resources []*models.KubernetesResource
	var resourcesMutex sync.Mutex
	resourceChan := make(chan *models.KubernetesResource, 1000)
	doneChan := make(chan bool, 1)
	resourceCount := atomic.Int32{}
	progressDone := make(chan bool, 1)
	if !options.QuietMode {
		go utils.PrintProgress(progressDone, "Collecting resources", options)
	}
	topics := []string{
		"meshery.meshsync.core",             
		"meshery.meshsync.core.resource",    
		"meshery.meshsync",                  
		"meshery.meshsync.resource",         
	}
	var subs []*nats.Subscription
	for _, topic := range topics {
		if options.VerboseMode {
			fmt.Printf("Subscribing to NATS topic: %s\n", topic)
		}
		sub, err := nc.Subscribe(topic, func(msg *nats.Msg) {
			var message struct {
				Object *models.KubernetesResource `json:"Object"`
				ObjectType string                 `json:"ObjectType"`
				EventType string                  `json:"EventType"`
			}
			if err := json.Unmarshal(msg.Data, &message); err != nil {
				var directResource models.KubernetesResource
				if err2 := json.Unmarshal(msg.Data, &directResource); err2 != nil {
					if options.VerboseMode {
						fmt.Printf("Warning: Could not unmarshal message: %v\n", err2)
					}
					return
				}
				resourceChan <- &directResource
				return
			}
			if message.Object != nil && (message.EventType == "" || message.EventType == "ADDED") {
				resourceChan <- message.Object
			}
		})
		if err != nil {
			if options.VerboseMode {
				fmt.Printf("Warning: Failed to subscribe to %s: %v\n", topic, err)
			}
			continue
		}
		subs = append(subs, sub)
	}
	if len(subs) == 0 {
		close(progressDone)
		return nil, fmt.Errorf("failed to subscribe to any NATS topics")
	}
	defer func() {
		for _, sub := range subs {
			sub.Unsubscribe()
		}
	}()
	go func() {
		time.Sleep(3 * time.Second)  
		ticker := time.NewTicker(300 * time.Millisecond)  
		defer ticker.Stop()
		lastCount := int32(0)
		stableCount := 0
		for {
			select {
			case <-ticker.C:
				currentCount := resourceCount.Load()
				if currentCount > 10 {
					if currentCount == lastCount {
						stableCount++
					} else {
						stableCount = 0
					}
					if stableCount >= 3 { 
						select {
						case doneChan <- true:
						default:
						}
						return
					}
				}
				lastCount = currentCount
			case <-ctx.Done():
				return
			}
		}
	}()
	collectionTimer := time.NewTimer(options.CollectionTime)
	defer collectionTimer.Stop()
	seenResourceKeys := make(map[string]bool)
	collecting := true
	go func() {
		for collecting {
			select {
			case resource := <-resourceChan:
				if resource != nil && resource.KubernetesResourceMeta != nil {
					resourcesMutex.Lock()
					key := fmt.Sprintf("%s/%s/%s", 
						resource.Kind, 
						resource.KubernetesResourceMeta.Namespace,
						resource.KubernetesResourceMeta.Name)
					if !seenResourceKeys[key] {
						seenResourceKeys[key] = true
						resources = append(resources, resource)
						resourceCount.Add(1)
					}
					resourcesMutex.Unlock()
				}
			case <-collectionTimer.C:
				select {
				case doneChan <- true:
				default:
				}
				return
			case <-ctx.Done():
				select {
				case doneChan <- true:
				default:
				}
				return
			}
		}
	}()
	<-doneChan
	collecting = false
	close(progressDone)
	filteredResources := utils.FilterResources(resources, options)
	if options.VerboseMode {
		fmt.Printf("Collected %d resources, filtered to %d resources\n", 
			len(resources), len(filteredResources))
	}
	return filteredResources, nil
}
func previewResources(options *models.Options) ([]*models.KubernetesResource, error) {
	sampleResources := []*models.KubernetesResource{
		{
			Kind: "Namespace",
			KubernetesResourceMeta: &models.KubernetesResourceObjectMeta{
				Name: "default",
			},
		},
		{
			Kind: "Pod",
			KubernetesResourceMeta: &models.KubernetesResourceObjectMeta{
				Name: "sample-pod",
				Namespace: "default",
			},
		},
		{
			Kind: "Deployment",
			KubernetesResourceMeta: &models.KubernetesResourceObjectMeta{
				Name: "sample-deployment",
				Namespace: "default",
			},
		},
	}
	return utils.FilterResources(sampleResources, options), nil
}