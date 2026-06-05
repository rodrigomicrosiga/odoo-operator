# Permite execução apenas no cluster local (Kind) para segurança
allow_k8s_contexts('kind-kind')

# Gera os códigos automáticos (DeepCopy) e os manifestos (CRDs, RBAC)
local('make manifests generate')

# Processa o Kustomize para gerar o YAML final de implantação
yaml = local('kustomize build config/default')
k8s_yaml(yaml)

# Faz o build da imagem do Operator em tempo real.
# O nome 'controller' é o padrão que o Kubebuilder v4 injeta no config/manager.
# O Tilt substitui essa tag on-the-fly pela imagem local compilada.
docker_build(
    'controller',
    '.',
    ignore=['bin/', 'config/', '.git/', 'hack/']
)

# Configura o recurso na UI do Tilt e expõe a porta de métricas
k8s_resource(
    'odoo-operator-controller-manager',
    port_forwards=['8080:8080']
)