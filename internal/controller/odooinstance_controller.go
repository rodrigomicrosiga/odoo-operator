package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloud104/reconciler/v2"
	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
	"github.com/rodrigomicrosiga/odoo-operator/internal/controller/odooinstance"
)

type OdooInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=odoo.cloud104.io,resources=odooinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=odoo.cloud104.io,resources=odooinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=odoo.cloud104.io,resources=odooinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;services;configmaps;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *OdooInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	inst := &v1alpha1.OdooInstance{}
	if err := r.Get(ctx, req.NamespacedName, inst); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Aciona a Chain of Responsibility
	chain := r.buildChain()
	return chain.Run(ctx, inst)
}

func (r *OdooInstanceReconciler) buildChain() reconciler.Chain[*v1alpha1.OdooInstance] {
	return reconciler.NewChain(
		&odooinstance.SecretEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.DatabaseStatefulSetEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.DatabaseServiceEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.OdooConfigMapEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.OdooFilestorePVCEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.OdooDeploymentEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.OdooServiceEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odooinstance.StatusEnsurer{Client: r.Client},
	)
}

func (r *OdooInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OdooInstance{}).
		Owns(&corev1.Secret{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
