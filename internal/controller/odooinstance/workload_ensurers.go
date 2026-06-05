package odooinstance

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/cloud104/reconciler/v2"
	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
	"github.com/rodrigomicrosiga/odoo-operator/internal/factory"
)

// ---------------------------------------------------------
// 1. Database StatefulSet Ensurer
// ---------------------------------------------------------
type DatabaseStatefulSetEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *DatabaseStatefulSetEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	desired := factory.BuildPostgresStatefulSet(inst)
	obj := &appsv1.StatefulSet{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		// Proteção fundamental de campos imutáveis da API do K8s
		if obj.CreationTimestamp.IsZero() {
			obj.Spec.VolumeClaimTemplates = desired.Spec.VolumeClaimTemplates
		}
		obj.Labels = desired.Labels
		obj.Spec.Replicas = desired.Spec.Replicas
		obj.Spec.Selector = desired.Spec.Selector
		obj.Spec.Template = desired.Spec.Template
		return controllerutil.SetControllerReference(inst, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, inst)
}

// ---------------------------------------------------------
// 2. Database Service Ensurer
// ---------------------------------------------------------
type DatabaseServiceEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *DatabaseServiceEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	desired := factory.BuildPostgresService(inst)
	obj := &corev1.Service{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		obj.Labels = desired.Labels
		obj.Spec.Selector = desired.Spec.Selector
		obj.Spec.Ports = desired.Spec.Ports
		return controllerutil.SetControllerReference(inst, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, inst)
}

// ---------------------------------------------------------
// 3. Odoo ConfigMap Ensurer
// ---------------------------------------------------------
type OdooConfigMapEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *OdooConfigMapEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	desired := factory.BuildOdooConfigMap(inst)
	obj := &corev1.ConfigMap{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		obj.Labels = desired.Labels
		obj.Data = desired.Data
		return controllerutil.SetControllerReference(inst, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, inst)
}

// ---------------------------------------------------------
// 4. Odoo Filestore PVC Ensurer
// ---------------------------------------------------------
type OdooFilestorePVCEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *OdooFilestorePVCEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	desired := factory.BuildOdooFilestorePVC(inst)
	obj := &corev1.PersistentVolumeClaim{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		// PVC Specs são imutáveis após a criação (exceto storage size)
		if obj.CreationTimestamp.IsZero() {
			obj.Spec = desired.Spec
		}
		obj.Labels = desired.Labels
		return controllerutil.SetControllerReference(inst, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, inst)
}

// ---------------------------------------------------------
// 5. Odoo Deployment Ensurer
// ---------------------------------------------------------
type OdooDeploymentEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *OdooDeploymentEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	desired := factory.BuildOdooDeployment(inst)
	obj := &appsv1.Deployment{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		obj.Labels = desired.Labels
		obj.Spec.Replicas = desired.Spec.Replicas
		obj.Spec.Selector = desired.Spec.Selector
		obj.Spec.Template = desired.Spec.Template
		return controllerutil.SetControllerReference(inst, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, inst)
}

// ---------------------------------------------------------
// 6. Odoo Service Ensurer
// ---------------------------------------------------------
type OdooServiceEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *OdooServiceEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	desired := factory.BuildOdooService(inst)
	obj := &corev1.Service{ObjectMeta: desired.ObjectMeta}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, obj, func() error {
		obj.Labels = desired.Labels
		obj.Spec.Selector = desired.Spec.Selector
		obj.Spec.Ports = desired.Spec.Ports
		return controllerutil.SetControllerReference(inst, obj, e.Scheme)
	})
	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}
	return e.Next(ctx, inst)
}
