package crds

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

type Manager struct {
	crdFilePath string
	applied     bool
}

func NewManager(crdFilePath string) *Manager {
	return &Manager{
		crdFilePath: crdFilePath,
		applied:     false,
	}
}

func (m *Manager) Apply() error {
	fmt.Println("Applying MeshSync CRDs...")
	
	// Create temp file with CRD content
	tmpFile, err := ioutil.TempFile("", "meshery-crds-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	m.crdFilePath = tmpFile.Name()
	
	if _, err := tmpFile.Write([]byte(crdContent)); err != nil {
		return fmt.Errorf("failed to write CRDs to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	
	// Apply CRDs
	cmd := exec.Command("kubectl", "apply", "-f", m.crdFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply CRDs: %w\nOutput: %s", err, output)
	}

	// Create meshery namespace
	namespaceYAML := `
apiVersion: v1
kind: Namespace
metadata:
  name: meshery
`
	tmpNamespaceFile, err := ioutil.TempFile("", "meshery-namespace-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpNamespaceFile.Name())
	
	if _, err := tmpNamespaceFile.Write([]byte(namespaceYAML)); err != nil {
		return fmt.Errorf("failed to write namespace YAML to temporary file: %w", err)
	}
	if err := tmpNamespaceFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	
	cmd = exec.Command("kubectl", "apply", "-f", tmpNamespaceFile.Name())
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply namespace: %w\nOutput: %s", err, output)
	}

	// Create Broker CR
	brokerYAML := `
apiVersion: meshery.io/v1alpha1
kind: Broker
metadata:
  name: meshery-broker
  namespace: meshery
spec:
  size: 1
`
	tmpBrokerFile, err := ioutil.TempFile("", "broker-instance-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpBrokerFile.Name())
	
	if _, err := tmpBrokerFile.Write([]byte(brokerYAML)); err != nil {
		return fmt.Errorf("failed to write Broker YAML to temporary file: %w", err)
	}
	if err := tmpBrokerFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	
	cmd = exec.Command("kubectl", "apply", "-f", tmpBrokerFile.Name())
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply Broker instance: %w\nOutput: %s", err, output)
	}

	// Wait a moment for broker to initialize
	fmt.Println("Waiting for broker to initialize...")
	cmd = exec.Command("sleep", "2")
	cmd.Run()

	// Create MeshSync CR with proper configuration
	meshSyncYAML := `
apiVersion: meshery.io/v1alpha1
kind: MeshSync
metadata:
  name: meshery-meshsync
  namespace: meshery
spec:
  broker:
    native:
      name: meshery-broker
      namespace: meshery
  size: 1
  watch-list:
    data:
      whitelist: '[{"Resource":"namespaces.v1.","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"configmaps.v1.","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"nodes.v1.","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"pods.v1.","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"services.v1.","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"deployments.v1.apps","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"statefulsets.v1.apps","Events":["ADDED","MODIFIED","DELETED"]},{"Resource":"daemonsets.v1.apps","Events":["ADDED","MODIFIED","DELETED"]}]'
`
	tmpMeshSyncFile, err := ioutil.TempFile("", "meshsync-instance-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpMeshSyncFile.Name())
	
	if _, err := tmpMeshSyncFile.Write([]byte(meshSyncYAML)); err != nil {
		return fmt.Errorf("failed to write MeshSync YAML to temporary file: %w", err)
	}
	if err := tmpMeshSyncFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	
	cmd = exec.Command("kubectl", "apply", "-f", tmpMeshSyncFile.Name())
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply MeshSync instance: %w\nOutput: %s", err, output)
	}
	
	m.applied = true
	fmt.Println("MeshSync CRDs and instance applied successfully")
	return nil
}

func (m *Manager) Remove() error {
	if !m.applied {
		return nil
	}
	
	fmt.Println("Removing MeshSync instance and CRDs...")
	
	// Remove MeshSync CR
	cmd := exec.Command("kubectl", "delete", "meshsync", "meshery-meshsync", "-n", "meshery", "--ignore-not-found=true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: failed to remove MeshSync instance: %v\nOutput: %s\n", err, output)
	}
	
	// Remove Broker CR
	cmd = exec.Command("kubectl", "delete", "broker", "meshery-broker", "-n", "meshery", "--ignore-not-found=true")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Warning: failed to remove Broker instance: %v\nOutput: %s\n", err, output)
	}
	
	// Remove CRDs
	if _, err := os.Stat(m.crdFilePath); err == nil {
		cmd = exec.Command("kubectl", "delete", "-f", m.crdFilePath, "--ignore-not-found=true")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Warning: failed to remove CRDs: %v\nOutput: %s\n", err, output)
		}
	}
	
	os.Remove(m.crdFilePath)
	m.applied = false
	fmt.Println("MeshSync instance and CRDs removed")
	return nil
}

// Continue with your existing crdContent variable
var crdContent = `
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.6.1
  name: brokers.meshery.io
spec:
  group: meshery.io
  names:
    kind: Broker
    listKind: BrokerList
    plural: brokers
    singular: broker
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: Broker is the Schema for the brokers API
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            description: BrokerSpec defines the desired state of Broker
            properties:
              size:
                format: int32
                type: integer
            type: object
          status:
            description: BrokerStatus defines the observed state of Broker
            properties:
              conditions:
                items:
                  properties:
                    lastProbeTime:
                      format: date-time
                      type: string
                    lastTransitionTime:
                      format: date-time
                      type: string
                    message:
                      type: string
                    observedGeneration:
                      format: int64
                      type: integer
                    reason:
                      type: string
                    status:
                      type: string
                    type:
                      type: string
                  required:
                  - lastTransitionTime
                  - message
                  - reason
                  - status
                  - type
                  type: object
                type: array
              endpoint:
                properties:
                  external:
                    type: string
                  internal:
                    type: string
                type: object
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.6.1
  name: meshsyncs.meshery.io
spec:
  group: meshery.io
  names:
    kind: MeshSync
    listKind: MeshSyncList
    plural: meshsyncs
    singular: meshsync
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: MeshSync is the Schema for the meshsyncs API
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            description: MeshSyncSpec defines the desired state of MeshSync
            properties:
              broker:
                properties:
                  custom:
                    properties:
                      url:
                        type: string
                    type: object
                  native:
                    properties:
                      name:
                        type: string
                      namespace:
                        type: string
                    type: object
                type: object
              version:  
                type: string
              size:
                format: int32
                type: integer
              watch-list:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      whitelist:
                        type: string
                      blacklist:
                        type: string
            type: object
          status:
            description: MeshSyncStatus defines the observed state of MeshSync
            properties:
              conditions:
                items:
                  properties:
                    lastProbeTime:
                      format: date-time
                      type: string
                    lastTransitionTime:
                      format: date-time
                      type: string
                    message:
                      type: string
                    observedGeneration:
                      format: int64
                      type: integer
                    reason:
                      type: string
                    status:
                      type: string
                    type:
                      type: string
                  required:
                  - lastTransitionTime
                  - message
                  - reason
                  - status
                  - type
                  type: object
                type: array
              publishing-to:
                type: string
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []`