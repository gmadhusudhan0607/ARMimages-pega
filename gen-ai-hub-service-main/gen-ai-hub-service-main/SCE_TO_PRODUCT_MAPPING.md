# SCE to Product Mapping

This document maps Service Catalog Entry (SCE) components to their corresponding products and resource types.

## Component to Product Mapping

| Component | Product | Resource Type | Subcomponents |
|-----------|---------|---------------|---------------|
| genai-hub-service | GenAIGatewayServiceProduct | backing-services | go:apidocs, go:cmd, go:internal, go:pkg, genai-hub-service-docker, genai-hub-service-sce, genai-hub-service-helm |
| role | GenAIGatewayServiceProduct | backing-services | role-sce, role-terraform |
| genai-awsbedrock-infra | GenAIInfrastructure | controlplane-services | genai-awsbedrock-infra-sce, genai-awsbedrock-infra-terraform |
| genai-defaults | GenAIInfrastructure | controlplane-services | genai-defaults-sce, genai-defaults-terraform |
| sax-iam-oidc-provider | GenAIInfrastructure | controlplane-services | sax-iam-oidc-provider-sce, sax-iam-oidc-provider-terraform |
| genai-gcpvertex-host | GenAIInfrastructureGCP | controlplane-services | genai-gcpvertex-host-sce, genai-gcpvertex-host-terraform |
| genai-gcpvertex-infra | GenAIInfrastructureGCP | controlplane-services | genai-gcpvertex-infra-sce, genai-gcpvertex-infra-terraform |
| genai-private-model-config | GenAIPrivateModels | backing-services | genai-private-model-config-sce, genai-private-model-config-terraform |
| genai-private-model-externalsecret | GenAIPrivateModels | backing-services | genai-private-model-externalsecret-sce, genai-private-model-externalsecret-helm |
| sax-registration | *(not found)* | *(not found)* | sax-registration-sce, sax-registration-terraform |

## Notes

- **go:** prefix indicates directories containing Go source code
- **SCE**: Service Catalog Entry (typically `*-sce` directories)
- **Resource Types**:
  - `backing-services`: Services that are deployed to clusters
  - `controlplane-services`: Infrastructure services deployed to control plane
- Product definitions are located in: `../gen-ai-gateway-service-product/src/main/resources/product-catalog/product-definitions/`
