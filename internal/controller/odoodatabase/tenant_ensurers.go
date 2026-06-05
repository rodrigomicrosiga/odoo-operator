package odoodatabase

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/cloud104/reconciler/v2"
	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
	"github.com/rodrigomicrosiga/odoo-operator/internal/factory"
)

// Usamos o Context do Go para transportar a Instância entre os Ensurers e evitar consultas repetitivas no K8s
type contextKey string

const instanceKey contextKey = "instance"

// ---------------------------------------------------------
// 1. O Guardião: ResolveInstanceEnsurer
// ---------------------------------------------------------
type ResolveInstanceEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooDatabase]
	Client client.Client
}

func (e *ResolveInstanceEnsurer) Reconcile(ctx context.Context, db *v1alpha1.OdooDatabase) (ctrl.Result, error) {
	inst := &v1alpha1.OdooInstance{}
	key := client.ObjectKey{Namespace: db.Namespace, Name: db.Spec.InstanceRef.Name}

	if err := e.Client.Get(ctx, key, inst); err != nil {
		// A instância não existe (ainda). Requeue sem lançar erro!
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if inst.Status.Phase != "Ready" {
		// A instância existe, mas o PostgreSQL e o App ainda estão subindo. Aguarda.
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Guarda a instância na memória da requisição para o próximo Ensurer
	ctx = context.WithValue(ctx, instanceKey, inst)
	return e.Next(ctx, db)
}

// ---------------------------------------------------------
// 2. O Inicializador: DatabaseInitJobEnsurer
// ---------------------------------------------------------
type DatabaseInitJobEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooDatabase]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *DatabaseInitJobEnsurer) Reconcile(ctx context.Context, db *v1alpha1.OdooDatabase) (ctrl.Result, error) {
	// Resgata a instância salva pelo Guardião
	inst := ctx.Value(instanceKey).(*v1alpha1.OdooInstance)

	job := &batchv1.Job{}
	name := client.ObjectKey{Namespace: db.Namespace, Name: db.Name + "-init"}

	err := e.Client.Get(ctx, name, job)
	if err == nil {
		// MAGIA DA IDEMPOTÊNCIA: O Job já existe! Nós simplesmente pulamos.
		return e.Next(ctx, db)
	}
	if !apierrors.IsNotFound(err) {
		return e.RequeueOnErr(ctx, err)
	}

	// O Job não existe, então criamos do zero.
	desired := factory.BuildInitJob(db, inst)
	if err := controllerutil.SetControllerReference(db, desired, e.Scheme); err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	if err := e.Client.Create(ctx, desired); err != nil {
		return e.RequeueOnErr(ctx, err)
	}

	return e.Next(ctx, db)
}

// ---------------------------------------------------------
// 3. O Roteador: IngressEnsurer
// ---------------------------------------------------------
type IngressEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooDatabase]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *IngressEnsurer) Reconcile(ctx context.Context, db *v1alpha1.OdooDatabase) (ctrl.Result, error) {
	inst := ctx.Value(instanceKey).(*v1alpha1.OdooInstance)
	desired := factory.BuildTenantIngress(db, inst)
	obj := &networkingv1.Ingress{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		obj.Labels = desired.Labels
		obj.Annotations = desired.Annotations
		obj.Spec = desired.Spec
		return controllerutil.SetControllerReference(db, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, db)
}

// ---------------------------------------------------------
// 4. O Observador: StatusEnsurer
// ---------------------------------------------------------
type StatusEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooDatabase]
	Client client.Client
}

func (e *StatusEnsurer) Reconcile(ctx context.Context, db *v1alpha1.OdooDatabase) (ctrl.Result, error) {
	setDBCondition(db, "InstanceReady", true, "OdooInstance resolvida e operante")

	jobReady := false
	job := &batchv1.Job{}
	if err := e.Client.Get(ctx, client.ObjectKey{Name: db.Name + "-init", Namespace: db.Namespace}, job); err == nil {
		if job.Status.Succeeded > 0 {
			jobReady = true
		}
	}
	setDBCondition(db, "DatabaseCreated", jobReady, "Job de criação do banco lógico concluído")

	if jobReady {
		db.Status.Phase = "Ready"
	} else {
		db.Status.Phase = "Creating"
	}
	db.Status.ObservedGeneration = db.Generation

	if err := e.Client.Status().Update(ctx, db); err != nil {
		return e.RequeueOnErr(ctx, err)
	}

	return e.Next(ctx, db)
}

func setDBCondition(db *v1alpha1.OdooDatabase, conditionType string, status bool, message string) {
	condStatus := metav1.ConditionFalse
	reason := "NotReady"
	if status {
		condStatus = metav1.ConditionTrue
		reason = "Ready"
	}
	meta.SetStatusCondition(&db.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             condStatus,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: db.Generation,
	})
}
