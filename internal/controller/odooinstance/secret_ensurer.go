package odooinstance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/cloud104/reconciler/v2"
	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
	"github.com/rodrigomicrosiga/odoo-operator/internal/factory"
)

// geraSenha cria uma string hexadecimal aleatória de 16 caracteres para ser usada como senha forte
func geraSenha() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback seguro em caso extremo de falha no crypto/rand
		return "fallback-odoo-admin-123"
	}
	return hex.EncodeToString(bytes)
}

type SecretEnsurer struct {
	reconciler.Funcs[*v1alpha1.OdooInstance]
	Client client.Client
	Scheme *runtime.Scheme
}

func (e *SecretEnsurer) Reconcile(ctx context.Context, inst *v1alpha1.OdooInstance) (ctrl.Result, error) {
	// Objeto "em branco" apontando para o nome/namespace que desejamos.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-secret",
			Namespace: inst.Namespace,
		},
	}

	// O CreateOrUpdate vai buscar o Secret no cluster. Se não existir, ele vai criar.
	// Se existir, ele vai baixar o estado atual para dentro da variável 'secret' antes de rodar a função Mutate.
	_, err := controllerutil.CreateOrUpdate(ctx, e.Client, secret, func() error {

		// 💡 A MÁGICA DA IDEMPOTÊNCIA ACONTECE AQUI:
		// Verificamos se o Secret tem o mapa Data vazio (ou seja, é um Secret virgem, recém-criado na memória).
		// Se for, geramos as senhas. Se não for, ele já tem dados e o Kubernetes irá preservá-los intocados!
		if secret.Data == nil && len(secret.StringData) == 0 {
			adminPassword := geraSenha()
			pgPassword := geraSenha()

			secret.StringData = map[string]string{
				"postgres-password": pgPassword,
				"odoo.conf":         fmt.Sprintf("[options]\nadmin_passwd = %s\n", adminPassword),
			}
		}

		// Atualizamos as labels garantindo que o Factory dite as regras padronizadas
		secret.Labels = factory.InstanceLabels(inst, "secret")

		// Amarramos o ciclo de vida deste Secret ao CRD OdooInstance (Garbage Collection)
		return controllerutil.SetControllerReference(inst, secret, e.Scheme)
	})

	if err != nil {
		return e.RequeueOnErr(ctx, err)
	}

	// Sucesso! Passa a bola para o próximo Ensurer na corrente.
	return e.Next(ctx, inst)
}
