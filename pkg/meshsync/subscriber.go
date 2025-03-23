package meshsync

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/utils"
	"github.com/nats-io/nats.go"
)

func CollectResources(ctx context.Context, natsURL string, options *models.Options) ([]*models.KubernetesResource, error) {
	// Don't actually connect to NATS in preview mode
	if options.PreviewMode {
		return previewResources(options)
	}
	
	// Connect to NATS with reconnection options
	nc, err := nats.Connect(natsURL, 
		nats.ReconnectWait(1*time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if !options.QuietMode {
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
	doneChan := make(chan bool)
	
	// Set up progress indicator
	progressDone := make(chan bool)
	if !options.QuietMode {
		go utils.PrintProgress(progressDone, "Collecting resources", options)
	}
	
	// Subscribe to multiple potential topics where MeshSync might publish
	topics := []string{
		"meshery.meshsync.core",             
		"meshery.meshsync.core.resource",    
		"meshsync.resources",                
		"meshery.meshsync.resources",        
	}
	
	var subs []*nats.Subscription
	
	// Subscribe to all topics
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
				// Try another format where object might be directly in the message
				var directResource models.KubernetesResource
				if err2 := json.Unmarshal(msg.Data, &directResource); err2 != nil {
					if options.VerboseMode {
						fmt.Printf("Warning: Failed to unmarshal message: %v\n", err)
					}
					return
				}
				resourceChan <- &directResource
				return
			}
			
			if message.Object != nil {
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
	
	// Cleanup function for subscriptions
	defer func() {
		for _, sub := range subs {
			sub.Unsubscribe()
		}
	}()
	
	// Set up collection timeout
	collectionTimer := time.NewTimer(options.CollectionTime)
	defer collectionTimer.Stop()
	
	// Collection loop with deduplicate logic
	seenResourceKeys := make(map[string]bool)
	collecting := true
	
	go func() {
		for collecting {
			select {
			case resource := <-resourceChan:
				if resource != nil {
					resourcesMutex.Lock()
					
					// Create a unique key for this resource to prevent duplicates
					key := fmt.Sprintf("%s/%s/%s", 
						resource.Kind, 
						resource.KubernetesResourceMeta.Namespace,
						resource.KubernetesResourceMeta.Name)
						
					if !seenResourceKeys[key] {
						seenResourceKeys[key] = true
						resources = append(resources, resource)
						
						if options.VerboseMode {
							fmt.Printf("Collected: %s\n", key)
						}
					}
					resourcesMutex.Unlock()
				}
			case <-collectionTimer.C:
				close(doneChan)
				return
			case <-ctx.Done():
				close(doneChan)
				return
			}
		}
	}()
	
	// Wait for completion
	<-doneChan
	collecting = false
	close(progressDone)
	
	// Apply filters
	filteredResources := utils.FilterResources(resources, options)
	
	if !options.QuietMode {
		fmt.Printf("Collected %d resources, filtered to %d resources\n", 
			len(resources), len(filteredResources))
	}
	
	return filteredResources, nil
}

// previewResources returns a sample set of resources for preview mode
func previewResources(options *models.Options) ([]*models.KubernetesResource, error) {
	// Create a sample set of resources
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

// package meshsync

// import (
// 	"encoding/json"
// 	"fmt"
// 	"time"

// 	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
// 	"github.com/nats-io/nats.go"
// )

// // func CollectResources(natsURL string, duration time.Duration) ([]*models.KubernetesResource, error) {
// // 	nc, err := nats.Connect(natsURL)
// // 	if err != nil {
// // 		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
// // 	}
// // 	defer nc.Close()

// // 	var resources []*models.KubernetesResource
// // 	resourceChan := make(chan *models.KubernetesResource, 1000)
// // 	doneChan := make(chan bool)

// // 	// Subscribe to multiple potential topics where MeshSync might publish
// // 	topics := []string{
// // 		"meshery.meshsync.core",              // Main topic
// // 		"meshery.meshsync.core.resource",     // Alternative topic
// // 		"meshsync.resources",                 // Another possible topic
// // 		"meshery.meshsync.resources",         // Yet another possibility
// // 	}

// // 	var subs []*nats.Subscription

// // 	// Subscribe to all topics
// // 	for _, topic := range topics {
// // 		fmt.Printf("Subscribing to NATS topic: %s\n", topic)
// // 		sub, err := nc.Subscribe(topic, func(msg *nats.Msg) {
// // 			var message struct {
// // 				Object *models.KubernetesResource `json:"Object"`
// // 				ObjectType string                 `json:"ObjectType"`
// // 			}

// // 			if err := json.Unmarshal(msg.Data, &message); err != nil {
// // 				// Try another format where object might be directly in the message
// // 				var directResource models.KubernetesResource
// // 				if err2 := json.Unmarshal(msg.Data, &directResource); err2 != nil {
// // 					fmt.Printf("Warning: Failed to unmarshal message: %v\n", err)
// // 					return
// // 				}
// // 				resourceChan <- &directResource
// // 				return
// // 			}

// // 			if message.Object != nil {
// // 				resourceChan <- message.Object
// // 			}
// // 		})

// // 		if err != nil {
// // 			fmt.Printf("Warning: Failed to subscribe to %s: %v\n", topic, err)
// // 			continue
// // 		}

// // 		subs = append(subs, sub)
// // 	}

// // 	if len(subs) == 0 {
// // 		return nil, fmt.Errorf("failed to subscribe to any NATS topics")
// // 	}

// // 	// Cleanup function for subscriptions
// // 	defer func() {
// // 		for _, sub := range subs {
// // 			sub.Unsubscribe()
// // 		}
// // 	}()

// // 	// Set a timeout
// // 	go func() {
// // 		time.Sleep(duration)
// // 		doneChan <- true
// // 	}()

// // 	// Print a heartbeat message to show progress
// // 	go func() {
// // 		ticker := time.NewTicker(5 * time.Second)
// // 		defer ticker.Stop()

// // 		for {
// // 			select {
// // 			case <-ticker.C:
// // 				fmt.Printf("Collected %d resources so far...\n", len(resources))
// // 			case <-doneChan:
// // 				return
// // 			}
// // 		}
// // 	}()

// // 	// Additionally, try to publish a ping message to see if broker is working
// // 	err = nc.Publish("meshery.meshsync.ping", []byte("ping"))
// // 	if err != nil {
// // 		fmt.Printf("Warning: Failed to publish ping message: %v\n", err)
// // 	}

// // 	// Collection loop with timeout
// // 	timeoutChan := time.After(duration)
// // 	collecting := true

// // 	for collecting {
// // 		select {
// // 		case resource := <-resourceChan:
// // 			resources = append(resources, resource)
// // 		case <-timeoutChan:
// // 			collecting = false
// // 		}
// // 	}

// // 	fmt.Printf("Collection complete. Found %d resources.\n", len(resources))

// // 	if len(resources) == 0 {
// // 		fmt.Println("No resources were collected. This might indicate an issue with MeshSync or the NATS connection.")
// // 		fmt.Println("Please check that MeshSync is running correctly and can connect to the NATS broker.")
// // 	}

// // 	return resources, nil
// // }

// func CollectResources(natsURL string, duration time.Duration) ([]*models.KubernetesResource, error) {
//     nc, err := nats.Connect(natsURL, nats.ReconnectWait(1*time.Second), nats.MaxReconnects(5))
//     if err != nil {
//         return nil, fmt.Errorf("failed to connect to NATS: %w", err)
//     }
//     defer nc.Close()
    
//     var resources []*models.KubernetesResource
//     resourceChan := make(chan *models.KubernetesResource, 1000)
//     doneChan := make(chan bool)
    
//     topics := []string{
//         "meshery.meshsync.core",
//         "meshery.meshsync.core.resource",
//         "meshsync.resources",
//     }
    
//     var subs []*nats.Subscription
    
//     for _, topic := range topics {
//         fmt.Printf("Subscribing to NATS topic: %s\n", topic)
//         sub, err := nc.Subscribe(topic, func(msg *nats.Msg) {
//             var message struct {
//                 Object *models.KubernetesResource `json:"Object"`
//                 ObjectType string                 `json:"ObjectType"`
//                 EventType string                  `json:"EventType"`
//             }
            
//             if err := json.Unmarshal(msg.Data, &message); err != nil {
//                 fmt.Printf("Warning: Failed to unmarshal message: %v\n", err)
//                 return
//             }
            
//             if message.Object != nil {
//                 resourceChan <- message.Object
//             }
//         })
        
//         if err != nil {
//             fmt.Printf("Warning: Failed to subscribe to %s: %v\n", topic, err)
//             continue
//         }
        
//         subs = append(subs, sub)
//     }
    
//     if len(subs) == 0 {
//         return nil, fmt.Errorf("failed to subscribe to any NATS topics")
//     }
    
//     defer func() {
//         for _, sub := range subs {
//             sub.Unsubscribe()
//         }
//     }()
    
//     timer := time.NewTimer(duration)
//     defer timer.Stop()
    
//     go func() {
//         ticker := time.NewTicker(5 * time.Second)
//         defer ticker.Stop()
        
//         for {
//             select {
//             case <-ticker.C:
//                 fmt.Printf("Collected %d resources so far...\n", len(resources))
//             case <-doneChan:
//                 return
//             }
//         }
//     }()
    
//     collecting := true
    
//     for collecting {
//         select {
//         case resource := <-resourceChan:
//             if resource != nil {
//                 resources = append(resources, resource)
//             }
//         case <-timer.C:
//             collecting = false
//             fmt.Printf("Collection complete. Found %d resources.\n", len(resources))
//         }
//     }
    
//     close(doneChan)
    
//     if len(resources) == 0 {
//         fmt.Println("No resources were collected. Trying to reconnect...")
//         time.Sleep(1 * time.Second)
//         return resources, nil
//     }
    
//     return resources, nil
// }