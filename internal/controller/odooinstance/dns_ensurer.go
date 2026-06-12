package odooinstance

import (
	"context"
	"fmt"

	"github.com/cloud104/reconciler/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
)

type OdooDNSEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *OdooDNSEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	// Se o domínio não estiver preenchido, pula silenciosamente para o próximo ensurer
	if inst.Spec.Domain == "" {
		return e.Next(ctx, inst)
	}

	svcName := fmt.Sprintf("%s-external-dns", inst.Name)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: inst.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, svc, func() error {
		if svc.Annotations == nil {
			svc.Annotations = make(map[string]string)
		}
		// Anotação obrigatória do padrão TCloud para o ExternalDNS
		svc.Annotations["external-dns.alpha.kubernetes.io/hostname"] = inst.Spec.Domain

		svc.Labels = inst.Labels
		svc.Spec.Type = corev1.ServiceTypeExternalName
		svc.Spec.ExternalName = inst.Spec.Domain

		return controllerutil.SetControllerReference(inst, svc, e.Scheme)
	})

	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}

	return e.Next(ctx, inst)
}
