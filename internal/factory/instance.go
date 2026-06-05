package factory

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1alpha1 "github.com/rodrigomicrosiga/odoo-operator/api/v1alpha1"
)

// Labels communs para rastreabilidade
func InstanceLabels(inst *v1alpha1.OdooInstance, component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "odoo",
		"app.kubernetes.io/instance":   inst.Name,
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/managed-by": "odoo-operator",
	}
}

// 1. Secret: Guarda a senha do DB e renderiza o odoo.conf com o admin_passwd
func BuildInstanceSecret(inst *v1alpha1.OdooInstance, pgPassword, adminPassword string) *corev1.Secret {
	odooConf := fmt.Sprintf("[options]\nadmin_passwd = %s\n", adminPassword)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-secret",
			Namespace: inst.Namespace,
			Labels:    InstanceLabels(inst, "secret"),
		},
		StringData: map[string]string{
			"postgres-password": pgPassword,
			"odoo.conf":         odooConf, // Arquivo físico que será montado
		},
	}
}

// 2. StatefulSet: O Banco de Dados PostgreSQL
func BuildPostgresStatefulSet(inst *v1alpha1.OdooInstance) *appsv1.StatefulSet {
	labels := InstanceLabels(inst, "database")
	replicas := int32(1)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-pg",
			Namespace: inst.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "postgres",
						Image: inst.Spec.Database.Image,
						Env: []corev1.EnvVar{
							{Name: "POSTGRES_USER", Value: inst.Spec.Database.User},
							{Name: "POSTGRES_DB", Value: "postgres"}, // Odoo usa o postgres para criar os tenants
							{
								Name: "POSTGRES_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{Name: inst.Name + "-secret"},
										Key:                  "postgres-password",
									},
								},
							},
						},
						Ports: []corev1.ContainerPort{{ContainerPort: 5432, Name: "postgres"}},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "pgdata", MountPath: "/var/lib/postgresql/data"},
						},
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{Name: "pgdata"},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: inst.Spec.Database.StorageSize},
					},
				},
			}},
		},
	}
}

// 3. Service: Apontamento para o PostgreSQL
func BuildPostgresService(inst *v1alpha1.OdooInstance) *corev1.Service {
	labels := InstanceLabels(inst, "database")
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-pg",
			Namespace: inst.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    []corev1.ServicePort{{Port: 5432, TargetPort: intstr.FromInt(5432)}},
		},
	}
}

// 4. ConfigMap: Configurações públicas do Odoo
func BuildOdooConfigMap(inst *v1alpha1.OdooInstance) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-cm",
			Namespace: inst.Namespace,
			Labels:    InstanceLabels(inst, "application"),
		},
		Data: map[string]string{
			"HOST":       inst.Name + "-pg",
			"PORT":       "5432",
			"USER":       inst.Spec.Database.User,
			"PROXY_MODE": "True", // Necessário quando roda atrás de um Ingress
			"LIST_DB":    "True",
		},
	}
}

// 5. PVC: O Filestore do Odoo (para guardar anexos e não perdê-los em restarts)
func BuildOdooFilestorePVC(inst *v1alpha1.OdooInstance) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-filestore",
			Namespace: inst.Namespace,
			Labels:    InstanceLabels(inst, "application"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: inst.Spec.Odoo.StorageSize},
			},
		},
	}
}

// 6. Deployment: A Aplicação Odoo
func BuildOdooDeployment(inst *v1alpha1.OdooInstance) *appsv1.Deployment {
	labels := InstanceLabels(inst, "application")

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-app",
			Namespace: inst.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &inst.Spec.Odoo.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "odoo",
						Image: inst.Spec.Odoo.Image,
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
						Ports: []corev1.ContainerPort{
							{ContainerPort: 8069, Name: "web"},
							{ContainerPort: 8072, Name: "longpolling"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "filestore", MountPath: "/var/lib/odoo"},
							{Name: "odoo-config", MountPath: "/etc/odoo"}, // Monta a pasta contendo o odoo.conf secreto
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

// 7. Service: Roteamento para a Aplicação
func BuildOdooService(inst *v1alpha1.OdooInstance) *corev1.Service {
	labels := InstanceLabels(inst, "application")
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      inst.Name + "-app",
			Namespace: inst.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Name: "web", Port: 8069, TargetPort: intstr.FromInt(8069)},
				{Name: "longpolling", Port: 8072, TargetPort: intstr.FromInt(8072)},
			},
		},
	}
}
