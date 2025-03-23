package models

type KubernetesResource struct {
	ID                     string                        `json:"id"`
	APIVersion             string                        `json:"apiVersion"`
	Kind                   string                        `json:"kind"`
	Model                  string                        `json:"model"`
	KubernetesResourceMeta *KubernetesResourceObjectMeta `json:"metadata"`
	Spec                   *KubernetesResourceSpec       `json:"spec,omitempty"`
	Status                 *KubernetesResourceStatus     `json:"status,omitempty"`
	ClusterID              string                        `json:"cluster_id"`
	PatternResource        interface{}                   `json:"pattern_resource"`
	ComponentMetadata      map[string]interface{}        `json:"component_metadata"`

	Immutable  string `json:"immutable,omitempty"`
	Data       string `json:"data,omitempty"`
	BinaryData string `json:"binaryData,omitempty"`
	StringData string `json:"stringData,omitempty"`
	Type       string `json:"type,omitempty"`
}

type KubernetesResourceObjectMeta struct {
	ID                         string                `json:"id"`
	Name                       string                `json:"name,omitempty"`
	GenerateName               string                `json:"generateName,omitempty"`
	Namespace                  string                `json:"namespace,omitempty"`
	SelfLink                   string                `json:"selfLink,omitempty"`
	UID                        string                `json:"uid"`
	ResourceVersion            string                `json:"resourceVersion,omitempty"`
	Generation                 int64                 `json:"generation,omitempty"`
	CreationTimestamp          string                `json:"creationTimestamp,omitempty"`
	DeletionTimestamp          string                `json:"deletionTimestamp,omitempty"`
	DeletionGracePeriodSeconds *int64                `json:"deletionGracePeriodSeconds,omitempty"`
	Labels                     []*KubernetesKeyValue `json:"labels,omitempty"`
	Annotations                []*KubernetesKeyValue `json:"annotations,omitempty"`
	OwnerReferences            string                `json:"ownerReferences,omitempty"`
	Finalizers                 string                `json:"finalizers,omitempty"`
	ClusterName                string                `json:"clusterName,omitempty"`
	ManagedFields              string                `json:"managedFields,omitempty"`
	ClusterID                  string                `json:"cluster_id"`
}

type KubernetesResourceSpec struct {
	ID        string `json:"id"`
	Attribute string `json:"attribute,omitempty"`
}

type KubernetesResourceStatus struct {
	ID        string `json:"id"`
	Attribute string `json:"attribute,omitempty"`
}

type KubernetesKeyValue struct {
	ID       string `json:"id"`
	UniqueID string `json:"unique_id"`
	Kind     string `json:"kind"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
}