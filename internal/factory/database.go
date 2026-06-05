package factory

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
)

// DatabaseLabels gera as labels padrão para os recursos do tenant
func DatabaseLabels(db *v1alpha1.OdooDatabase, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "odoo-tenant",
		"app.kubernetes.io/instance":   db.Name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/managed-by": "odoo-operator",
	}
}

// 1. Job: Executa a inicialização do banco lógico (equivalente a uma rotina de importação/setup)
func BuildInitJob(db *v1alpha1.OdooDatabase, inst *v1alpha1.OdooInstance) *batchv1.Job {
	labels := DatabaseLabels(db, "init-job")

	// Argumentos nativos do Odoo para criar o banco de dados via CLI
	args := []string{"odoo", "-d", db.Spec.DbName, "-i", "base", "--stop-after-init"}

	if db.Spec.Lang != "" {
		args = append(args, "--language", db.Spec.Lang)
	}
	if db.Spec.DemoData {
		args = append(args, "--without-demo=False")
	} else {
		args = append(args, "--without-demo=all")
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-init",
			Namespace: db.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever, // Jobs não devem reiniciar eternamente
					Containers: []corev1.Container{{
						Name:    "odoo-init",
						Image:   inst.Spec.Odoo.Image,
						Command: args,
						// Reutilizamos toda a injeção de ambiente que o servidor já usa!
						EnvFrom: []corev1.EnvFromSource{
							{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: inst.Name + "-cm"}}},
						},
						Env: []corev1.EnvVar{
							{
								Name: "PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: inst.Name + "-secret"},
										Key:                  "postgres-password",
									},
								},
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "filestore", MountPath: "/var/lib/odoo"},
							{Name: "odoo-config", MountPath: "/etc/odoo"},
						},
					}},
					Volumes: []corev1.Volume{
						{Name: "filestore", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: inst.Name + "-filestore"}}},
						{Name: "odoo-config", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: inst.Name + "-secret"}}},
					},
				},
			},
		},
	}
}

// 2. Ingress: Expõe a aplicação exclusivamente para este tenant
func BuildTenantIngress(db *v1alpha1.OdooDatabase, inst *v1alpha1.OdooInstance) *networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix

	// O dbfilter no header injetado no Nginx garante o isolamento lógico do tenant
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/server-snippet": `proxy_set_header X-Odoo-dbfilter "` + db.Spec.DbName + `";`,
	}

	var ingressClass *string
	if inst.Spec.IngressClassName != "" {
		ingressClass = &inst.Spec.IngressClassName
	}

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        db.Name + "-ingress",
			Namespace:   db.Namespace,
			Labels:      DatabaseLabels(db, "ingress"),
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: ingressClass,
			Rules: []networkingv1.IngressRule{
				{
					Host: db.Spec.Domain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: inst.Name + "-app",
											Port: networkingv1.ServiceBackendPort{Number: 8069},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
