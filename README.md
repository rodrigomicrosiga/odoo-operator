# 🐘 Odoo Operator for Kubernetes

Este repositório contém a implementação de um **Kubernetes Operator** multi-tenant desenvolvido para gerenciar o ERP open-source Odoo. O projeto utiliza o padrão **Chain of Responsibility** para orquestrar a infraestrutura compartilhada (PostgreSQL, Filestore, App) e o provisionamento isolado de bancos de dados lógicos para múltiplos clientes.

---

## 🏗️ Arquitetura e Fluxo de Reconciliação (Multi-Tenant)

O ecossistema é dividido em dois Custom Resources (CRDs) independentes, mas correlacionados:

```mermaid
graph TD
    subgraph CRD 1: OdooInstance
        I((Criar CRD<br>OdooInstance)) --> IC{Instance Controller}
        IC --> I_Sec[Secret: Senhas K8s]
        IC --> I_PG[StatefulSet: PostgreSQL]
        IC --> I_PVC[PVC: Filestore]
        IC --> I_CM[ConfigMap: Odoo Env]
        IC --> I_Dep[Deployment: Odoo App]
        I_Dep --> I_Svc[Service: 8069 / 8072]
        I_Svc --> I_R((Instance Ready))
    end

    subgraph CRD 2: OdooDatabase
        D((Criar CRD<br>OdooDatabase)) --> DC{Database Controller}
        DC -. "1. Watches & Resolve" .-> I_R
        DC --> D_Job[Job: Cria Banco Lógico]
        DC --> D_Ing[Ingress: Rota com dbfilter]
        D_Ing --> D_R((Tenant Ready))
    end

    style I fill:#3498db,stroke:#2980b9,stroke-width:2px,color:#fff
    style D fill:#9b59b6,stroke:#8e44ad,stroke-width:2px,color:#fff
    style I_R fill:#2ecc71,stroke:#27ae60,stroke-width:2px,color:#fff
    style D_R fill:#2ecc71,stroke:#27ae60,stroke-width:2px,color:#fff
```

📋 Pré-requisitos
Go (v1.24+)

Docker

kubectl

Kind (para criação do cluster local)

kubebuilder v4

🚀 Como Executar e Simular (Em Breve)\

(Os comandos de implantação, port-forward e simulação de criação/destruição serão documentados aqui conforme o avanço do projeto).

⚠️ Problemas Conhecidos e Soluções (Troubleshooting)\

(Espaço reservado para documentar os desafios de infraestrutura e código encontrados durante a jornada).

* [05/06/2026] - Setup Inicial e Planejamento:
  * Leitura e assimilação do escopo (`Odoo multi-tenant`).
  * Criação do repositório `odoo-operator` e definição do padrão arquitetural da documentação.
  * Entendimento da necessidade de dois CRDs (`OdooInstance` e `OdooDatabase`) e dos conceitos de `Referência Cruzada`, `Watches()` e `Idempotência via Kubernetes Jobs`.