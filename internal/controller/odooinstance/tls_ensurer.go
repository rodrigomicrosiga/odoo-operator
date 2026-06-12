package odooinstance

import (
	"context"
	"fmt"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cloud104/reconciler/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
)

type OdooTLSEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *OdooTLSEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	// Se o domínio não estiver preenchido, pula silenciosamente para o próximo ensurer
	if inst.Spec.Domain == "" {
		return e.Next(ctx, inst)
	}

	certName := fmt.Sprintf("%s-tls-secret", inst.Name)
	cert := &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: inst.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, cert, func() error {
		cert.Labels = inst.Labels
		cert.Spec = cmv1.CertificateSpec{
			SecretName: certName,
			DNSNames:   []string{inst.Spec.Domain},
			IssuerRef: cmmetav1.ObjectReference{
				Name: "letsencrypt-prod",
				Kind: "ClusterIssuer",
			},
		}

		return controllerutil.SetControllerReference(inst, cert, e.Scheme)
	})

	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}

	return e.Next(ctx, inst)
}
