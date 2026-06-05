package controller

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/cloud104/reconciler/v2"
	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
	"github.com/rodrigomicrosiga/odoo-operator/internal/controller/odoodatabase"
)

type OdooDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=odoo.cloud104.io,resources=odoodatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=odoo.cloud104.io,resources=odoodatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=odoo.cloud104.io,resources=odoodatabases/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *OdooDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	db := &v1alpha1.OdooDatabase{}
	if err := r.Get(ctx, req.NamespacedName, db); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	chain := r.buildChain()
	return chain.Reconcile(ctx, db)
}

func (r *OdooDatabaseReconciler) buildChain() reconciler.Handler[*v1alpha1.OdooDatabase] {
	return reconciler.Chain(
		&odoodatabase.ResolveInstanceEnsurer{Client: r.Client},
		&odoodatabase.DatabaseInitJobEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odoodatabase.IngressEnsurer{Client: r.Client, Scheme: r.Scheme},
		&odoodatabase.StatusEnsurer{Client: r.Client},
	)
}

func (r *OdooDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OdooDatabase{}).
		Owns(&batchv1.Job{}).
		Owns(&networkingv1.Ingress{}).
		// WATCHES CROSS-KIND: Escuta as mudanças no Servidor e aciona os Tenants dependentes
		Watches(
			&v1alpha1.OdooInstance{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
				inst := o.(*v1alpha1.OdooInstance)
				var dbList v1alpha1.OdooDatabaseList

				// Busca todos os Tenants no mesmo namespace...
				if err := r.List(ctx, &dbList, client.InNamespace(inst.Namespace)); err != nil {
					return nil
				}

				// ... e acorda apenas aqueles que apontam para o Servidor que mudou de status!
				var reqs []reconcile.Request
				for _, db := range dbList.Items {
					if db.Spec.InstanceRef.Name == inst.Name {
						reqs = append(reqs, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      db.Name,
								Namespace: db.Namespace,
							},
						})
					}
				}
				return reqs
			}),
		).
		Complete(r)
}
