package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanceReference aponta para o OdooInstance pai
type InstanceReference struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// OdooDatabaseSpec define o estado desejado do banco lógico
type OdooDatabaseSpec struct {
	// +kubebuilder:validation:Required
	InstanceRef InstanceReference `json:"instanceRef"`

	// DbName é o nome do banco de dados no PostgreSQL
	// +kubebuilder:validation:Required
	DbName string `json:"dbName"`

	// Domain é a URL exclusiva deste tenant
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`

	// +optional
	Lang string `json:"lang,omitempty"`

	// +optional
	DemoData bool `json:"demoData,omitempty"`
}

// OdooDatabaseStatus define o estado observado da criação do banco
type OdooDatabaseStatus struct {
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Instance",type=string,JSONPath=`.spec.instanceRef.name`
// +kubebuilder:printcolumn:name="DB Name",type=string,JSONPath=`.spec.dbName`
// +kubebuilder:printcolumn:name="Domain",type=string,JSONPath=`.spec.domain`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// OdooDatabase is the Schema for the odoodatabases API
type OdooDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OdooDatabaseSpec   `json:"spec,omitempty"`
	Status OdooDatabaseStatus `json:"status,omitempty"`
}

// ... (Mantenha o restante do arquivo OdooDatabaseList e init sem alterações)
