package odooinstance

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloud104/reconciler/v2"
	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
)

type StatusEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
}

func (e *StatusEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	dbReady := false
	appReady := false

	// 1. Checa o estado do StatefulSet do PostgreSQL
	sts := &appsv1.StatefulSet{}
	if err := e.Client.Get(ctx, client.ObjectKey{Name: inst.Name + "-pg", Namespace: inst.Namespace}, sts); err == nil {
		if sts.Status.ReadyReplicas > 0 {
			dbReady = true
		}
	}

	// 2. Checa o estado do Deployment do Odoo
	dep := &appsv1.Deployment{}
	if err := e.Client.Get(ctx, client.ObjectKey{Name: inst.Name + "-app", Namespace: inst.Namespace}, dep); err == nil {
		if dep.Status.ReadyReplicas == inst.Spec.Odoo.Replicas && inst.Spec.Odoo.Replicas > 0 {
			appReady = true
		}
	}

	// 3. Atualiza as Conditions
	setCondition(inst, "DatabaseReady", dbReady, "PostgreSQL está rodando e pronto")
	setCondition(inst, "AppReady", appReady, "Pods do Odoo estão rodando e prontos")

	isFullyReady := dbReady && appReady
	setCondition(inst, "Ready", isFullyReady, "Servidor OdooInstance totalmente provisionado")

	// 4. Atualiza a Phase e a URL
	if isFullyReady {
		inst.Status.Phase = "Ready"
		inst.Status.URL = "http://" + inst.Spec.Domain
	} else {
		inst.Status.Phase = "Provisioning"
	}
	inst.Status.ObservedGeneration = inst.Generation

	// Salva o status de volta no Kubernetes
	if err := e.Client.Status().Update(ctx, inst); err != nil {
		return e.RequeueOnErr(ctx, err)
	}

	return e.Next(ctx, inst)
}

// Função auxiliar genérica para facilitar a criação de Conditions
func setCondition(inst *v1alpha1.OdooInstance, conditionType string, status bool, message string) {
	condStatus := metav1.ConditionFalse
	reason := "NotReady"
	if status {
		condStatus = metav1.ConditionTrue
		reason = "Ready"
	}

	meta.SetStatusCondition(&inst.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             condStatus,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: inst.Generation,
	})
}
