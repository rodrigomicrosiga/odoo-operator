package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OdooAppSpec define os parâmetros do container do Odoo
type OdooAppSpec struct {
	// +kubebuilder:default:="odoo:17.0"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum:=0
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// +kubebuilder:default:="5Gi"
	// +optional
	StorageSize resource.Quantity `json:"storageSize,omitempty"`
}

// DatabaseSpec define os parâmetros do container do PostgreSQL
type DatabaseSpec struct {
	// +kubebuilder:default:="postgres:16"
	// +optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:default:="odoo"
	// +optional
	User string `json:"user,omitempty"`

	// +kubebuilder:default:="5Gi"
	// +optional
	StorageSize resource.Quantity `json:"storageSize,omitempty"`
}

// OdooInstanceSpec define o estado desejado pelo usuário
type OdooInstanceSpec struct {
	// Domain é o host base do servidor (obrigatório)
	// +kubebuilder:validation:Required
	Domain string `json:"domain"`

	// +optional
	Odoo OdooAppSpec `json:"odoo,omitempty"`

	// +optional
	Database DatabaseSpec `json:"database,omitempty"`

	// +optional
	IngressClassName string `json:"ingressClassName,omitempty"`
}

// OdooInstanceStatus define o estado real observado pelo Operator
type OdooInstanceStatus struct {
	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	URL string `json:"url,omitempty"`

	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`

// OdooInstance is the Schema for the odooinstances API
type OdooInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OdooInstanceSpec   `json:"spec,omitempty"`
	Status OdooInstanceStatus `json:"status,omitempty"`
}

// ... (Mantenha o restante do arquivo OdooInstanceList e init sem alterações)
//+kubebuilder:object:root=true

// OdooInstanceList contains a list of OdooInstance
type OdooInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OdooInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OdooInstance{}, &OdooInstanceList{})
}
