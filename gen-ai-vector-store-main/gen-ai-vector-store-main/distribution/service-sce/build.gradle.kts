/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */


import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild
import com.pega.gradle.plugin.model.DeploymentRefConfig

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "Deployment template to provision genai-vector-store on k8s cluster"

val serviceAuthenticationClientServiceVersion: String by project
val dbInstanceVersion: String by project
val helmVersion: String by project
val dbtoolsVersion: String by project

dependencies {
    assets(project(":distribution:service-helm", "archives"))
    services(project(":distribution:role-sce"))
    services(project(":distribution:service-infrastructure-sce"))

    services(group="com.pega.provisioning.services", name="DBInstance", version=dbInstanceVersion)
    services(group="com.pega.provisioning.services", name="ServiceAuthenticationClientService", version=serviceAuthenticationClientServiceVersion)
}

evaluationDependsOn(":distribution:service-docker")

val serviceNamespace: String by project

val dkBuildImage = project(":distribution:service-docker").tasks.named<DockerImageBuild>("buildImage")
val dockerRepo = dkBuildImage.get().image.name.orNull

val opsDkBuildImage = project(":distribution:ops-docker").tasks.named<DockerImageBuild>("buildImage")
val opsDockerRepo = opsDkBuildImage.get().image.name.orNull

val bkgDkBuildImage = project(":distribution:background-docker").tasks.named<DockerImageBuild>("buildImage")
val bkgDockerRepo = bkgDkBuildImage.get().image.name.orNull

sar {
    name = "GenAIVectorStore"
    description = "GenAI Vector Store"

    dynamicParam(
        "ProvisioningType",
        "Determines how many prompts are shown. Standard option will show fewer prompts compared to Advanced. Default = Standard"
    )
    dynamicParam("ClusterGUID", "GUID of CloudK cluster where service is deployed")
    dynamicParam("Namespace", "K8s namespace in which the service is created")
    dynamicParam("IsFrontendService", "Set to true to create ingress resources and to allow the service to get traffic from external client browsers")
    dynamicParam("LogLevel", "(Optional) Log level")
    dynamicParam("LogPerformanceTrace", "Set to true to output to service log the vector store performance trace")
    dynamicParam("EnableDBTools", "Set to true to create DB Tools for database debugging purposes")
    dynamicParam("EmbeddingModel", "The default model profile that will be used to embed the document chunks.")
    dynamicParam("UseOtlp", "Decides whether to use OTLP or not.")
    dynamicParam("OtlpSampler","Opentelemetry sampler (Defaults to parentbased_traceidratio)")
    dynamicParam("OtlpSamplerArg","Sampler argument value, ranges from 0 to 1 (Defaults to 0.1, means 10%)")
    dynamicParam("QueryEmbeddingTimeoutMs", "Query chunks/documents embedding timeout")
    dynamicParam("QueryEmbeddingMaxRetries", "Query chunks/documents embedding max retries")
    dynamicParam("UseLegacyAttributesIDs ", "Use Legacy Attributes IDs")
    dynamicParam("LogServiceMetrics", "Set to 'true' to enable logging of service metrics.")
    dynamicParam("UsageMetricsEnabled", "Set to 'true' to enable sending usage metrics to PDC.")

    // SAX JWT token caching parameters
    dynamicParam("SaxTokenCacheEnabled", "Enable JWT token caching to improve performance. Default is true")
    dynamicParam("SaxTokenCacheMaxTTL", "Maximum cache TTL for JWT tokens (security limit). Default is 50m")
    dynamicParam("SaxTokenCacheMaxSize", "Maximum number of JWT tokens to cache. Default is 10000")
    dynamicParam("SaxTokenCacheCleanupInterval", "Interval for cleaning up expired cache entries. Default is 5m")

    dynamicParam("IsolationIdVerificationDisabled", "Disable Isolation ID verification for SAX tokens. Default is false")

    fixedParam("service-base.saxTokenCacheEnabled", "dynamicParams.get('SaxTokenCacheEnabled')")
    fixedParam("service-base.saxTokenCacheMaxTTL", "dynamicParams.get('SaxTokenCacheMaxTTL')")
    fixedParam("service-base.saxTokenCacheMaxSize", "dynamicParams.get('SaxTokenCacheMaxSize')")
    fixedParam("service-base.saxTokenCacheCleanupInterval", "dynamicParams.get('SaxTokenCacheCleanupInterval')")

    fixedParam("service-base.dbEngineVersion", """
       ({ ->
          def cloudProvider = cmdbService.labels().get('provider')
          def dbVersion = ""
          if (cloudProvider == 'gcp') { 
              dbVersion = cmdbService.get('DBVersion')
          } else { 
              dbVersion = cmdbService.get('DatabaseEngineVersion')
          } 
          return dbVersion
       })()""".trimMargin())

    // Tests, debugging and troubleshooting purposes
    dynamicParam("RandomEmbedderEnabled", "Enable Random Embedder for testing purposes. Default is false")
    dynamicParam("RandomEmbedderDelay", "Delay in seconds for Random Embedder to simulate slow embedding. Default is 0 (no delay)")
    dynamicParam("ProfilerEnabled", "Enable Profiler for testing purposes. Default is false")
    dynamicParam("EncourageSemSearchIndexUse", "Encourage the use of semantic search index for queries. Default is false")
    fixedParam("service-base.randomEmbedderEnabled", "dynamicParams.get('RandomEmbedderEnabled')")
    fixedParam("service-base.randomEmbedderDelay", "dynamicParams.get('RandomEmbedderDelay')")
    fixedParam("service-base.profilerEnabled", "dynamicParams.get('ProfilerEnabled')")
    fixedParam("service-base.encourageSemSearchIndexUse", "dynamicParams.get('EncourageSemSearchIndexUse')")
    fixedParam("service-base.queryEmbeddingTimeoutMs", "dynamicParams.get('QueryEmbeddingTimeoutMs')")
    fixedParam("service-base.queryEmbeddingMaxRetries", "dynamicParams.get('QueryEmbeddingMaxRetries')")
    fixedParam("service-base.useLegacyAttributesIDs", "dynamicParams.get('UseLegacyAttributesIDs')")
    fixedParam("service-base.logServiceMetrics", "dynamicParams.get('LogServiceMetrics')")
    fixedParam("service-base.usageMetricsEnabled", "dynamicParams.get('UsageMetricsEnabled')")

    fixedParam("kubeconfig","cmdbService.cluster(dynamicParams.ClusterGUID).get('kubeconfig')")
    fixedParam("AccountID","cmdbService.cluster(dynamicParams.ClusterGUID).get('AccountID')")
    fixedParam("Region","cmdbService.cluster(dynamicParams.ClusterGUID).get('Region')")
    fixedParam("namespace","dynamicParams.Namespace ?: cmdbService.find('Namespace')")
    fixedParam("service-base.ClusterName","cmdbService.cluster(dynamicParams.ClusterGUID).get('ClusterID')")
    fixedParam("GenAIURL", "(cmdbService.findBackingService('clusterguid=' + dynamicParams.ClusterGUID, 'type=GenAI Gateway Service Product').find('GenAIHubServiceBaseURL')) ?: 'NotConfigured'")
    fixedParam("service-base.logLevel", "dynamicParams.get('LogLevel')?.trim()")
    fixedParam("service-base.logPerformanceTrace", "dynamicParams.get('LogPerformanceTrace')?.trim()")
    fixedParam("service-base.defaultEmbeddingProfile", "dynamicParams.get('EmbeddingModel')?.trim()")
    fixedParam("service-base.readOnlyMode", "cmdbService.isDeploymentActive() ? 'false' : 'true'")

    // OpenTelemetry Endpoint and Sampling configuration, it should be used when OpenTelemetry tracing library is used
    // Default value for sampler argument is 0.1, it can be overridden as following in service
    fixedParam("service-base.OtlpEndpoint","cmdbService.cluster(dynamicParams.ClusterGUID).find('OtlpEndpoint')")
    fixedParam("service-base.OtlpSampler","dynamicParams.OtlpSampler")
    fixedParam("service-base.OtlpSamplerArg","dynamicParams.OtlpSamplerArg")
    fixedParam("service-base.SamplingPercentage","dynamicParams.OtlpSamplerArg != null ? String.valueOf(Double.parseDouble(dynamicParams.OtlpSamplerArg) * 100) : String.valueOf(0.1 * 100)")

    //SAX
    fixedParam("service-base.SaxIssuer", "cmdbService.get('SaxIssuer')")
    fixedParam("service-base.SaxJWKSEndpoint", "cmdbService.get('SaxJWKSEndpoint')")
    fixedParam("service-base.SaxAudience", "cmdbService.get('SaxAudience')")

    fixedParam("service-base.isolationIdVerificationDisabled", "dynamicParams.get('IsolationIdVerificationDisabled')")
    // flags to expose the service outside the cluster. Disable by default
    fixedParam("isFrontendService","dynamicParams.get('IsFrontendService')?.trim()")
    fixedParam("service-base.LogPerformanceTrace","dynamicParams.get('LogPerformanceTrace')?.trim()")


    // VS service
    fixedParam("service-base.pegaservices.genai-vector-store.image.tag","'${project.version}'")
    fixedParam("service-base.pegaservices.genai-vector-store.image.repository","""
       ({ ->
          def cpStage = cmdbService.labels().get('cpStageName')
          if (cpStage == 'rnd-usgov') { 
              cpStage = 'artifactory-pcfrrd-cp-pl-endpoint.rnd-pcfrpegaservice.net:5000/${dockerRepo}'  
          } else if (cpStage == 'production-usgov') { 
              cpStage = 'artifactory-pcfr-cp-pl-endpoint.pcfrpegaservice.net:5000/${dockerRepo}' 
          } else if (['development', 'integration', 'staging', 'trials'].contains(cpStage)) { 
              cpStage = 'cirrus-docker.jfrog.io/${dockerRepo}' 
          } else { 
              cpStage = 'cirrus-docker-release.jfrog.io/${dockerRepo}'
          } 
          return cpStage
       })()""".trimMargin())
    fixedParam("service-base.pegaservices.genai-vector-store.serviceIRSARole","cmdbService.get('ServiceAccountRole')")
    fixedParam("service-base.pegaservices.genai-vector-store.egressRules","'tcp:' + cmdbService.get('DatabaseHost') + ':' + cmdbService.get('DatabasePort')",  "String" , "aws")
    fixedParam("service-base.pegaservices.genai-vector-store.egressRules","'tcp:' + cmdbService.get('DatabaseHost') + ':3307'",  "String" , "gcp")


    // Ops Service
    fixedParam("service-base.pegaservices.genai-vector-store-ops.image.tag","'${project.version}'")
    fixedParam("service-base.pegaservices.genai-vector-store-ops.image.repository","""
       ({ ->
          def cpStage = cmdbService.labels().get('cpStageName')
          if (cpStage == 'rnd-usgov') { 
              cpStage = 'artifactory-pcfrrd-cp-pl-endpoint.rnd-pcfrpegaservice.net:5000/${opsDockerRepo}'  
          } else if (cpStage == 'production-usgov') { 
              cpStage = 'artifactory-pcfr-cp-pl-endpoint.pcfrpegaservice.net:5000/${opsDockerRepo}' 
          } else if (['development', 'integration', 'staging', 'trials'].contains(cpStage)) { 
              cpStage = 'cirrus-docker.jfrog.io/${opsDockerRepo}' 
          } else { 
              cpStage = 'cirrus-docker-release.jfrog.io/${opsDockerRepo}'
          } 
          return cpStage
       })()""".trimMargin())
    fixedParam("service-base.pegaservices.genai-vector-store-ops.egressRules","'tcp:' + cmdbService.get('DatabaseHost') + ':' + cmdbService.get('DatabasePort')",  "String" , "aws")
    fixedParam("service-base.pegaservices.genai-vector-store-ops.egressRules","'tcp:' + cmdbService.get('DatabaseHost') + ':3307'",  "String" , "gcp")
    fixedParam("service-base.pegaservices.genai-vector-store-ops.serviceIngressPrefix","'/'.concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops/')")

    // Background Service
    fixedParam("service-base.pegaservices.genai-vector-store-background.image.tag", "'${project.version}'")
    fixedParam("service-base.pegaservices.genai-vector-store-background.image.repository","""
       ({ ->
          def cpStage = cmdbService.labels().get('cpStageName')
          if (cpStage == 'rnd-usgov') { 
              cpStage = 'artifactory-pcfrrd-cp-pl-endpoint.rnd-pcfrpegaservice.net:5000/${bkgDockerRepo}'  
          } else if (cpStage == 'production-usgov') { 
              cpStage = 'artifactory-pcfr-cp-pl-endpoint.pcfrpegaservice.net:5000/${bkgDockerRepo}' 
          } else if (['development', 'integration', 'staging', 'trials'].contains(cpStage)) { 
              cpStage = 'cirrus-docker.jfrog.io/${bkgDockerRepo}' 
          } else { 
              cpStage = 'cirrus-docker-release.jfrog.io/${bkgDockerRepo}'
          } 
          return cpStage
       })()""".trimMargin())
    fixedParam("service-base.pegaservices.genai-vector-store-background.egressRules","'tcp:' + cmdbService.get('DatabaseHost') + ':' + cmdbService.get('DatabasePort')",  "String" , "aws")
    fixedParam("service-base.pegaservices.genai-vector-store-background.egressRules","'tcp:' + cmdbService.get('DatabaseHost') + ':3307'",  "String" , "gcp")

    // SAX
    fixedParam("service-base.SaxClientID", "cmdbService.get('ServAuthClientID')")
    fixedParam("service-base.SaxClientScopes", "cmdbService.get('ServAuthScopes')")
    fixedParam("service-base.SaxClientSecret", "cmdbService.get('ServAuthSecretARN')")
    fixedParam("service-base.SaxClientTokenEndpoint", "cmdbService.get('ServAuthTokenEndpoint')")

    fixedParam("service-base.SaxOpsIssuer", "cmdbService.get('SaxOpsIssuer')")
    fixedParam("service-base.SaxOpsJWKSEndpoint", "cmdbService.get('SaxOpsJWKSEndpoint')")
    fixedParam("service-base.SaxOpsAudience", "cmdbService.get('SaxOpsAudience')")

    // Environment variables
    // cloudProvider is required to create proper SA role (by service-base)
    fixedParam("service-base.cloudProvider", "cmdbService.labels().get('provider')")
    fixedParam("service-base.account","cmdbService.labels().get('accountid')")
    fixedParam("service-base.region", "cmdbService.cluster(dynamicParams.ClusterGUID).get('Region')")
    fixedParam("service-base.databaseID","cmdbService.get('DatabaseID')")
    fixedParam("service-base.databaseName", "cmdbService.get('DatabaseName')")
    fixedParam("service-base.databaseHost", "cmdbService.get('DatabaseHost')")
    fixedParam("service-base.databasePort", "cmdbService.get('DatabasePort')")
    //Workaround for retrieving the DatabaseSecret in the ReadOnlyMode by reading it from ActiveRegion sce outputs - until US-641628 is completed on DBMS side.
    //fixedParam("service-base.databaseSecret", "(cmdbService.isDeploymentActive() ? cmdbService.get('DatabaseSecret') : (cmdbService.activeResource().get('DatabaseSecret')))")
    fixedParam("service-base.databaseSecret", "cmdbService.get('DatabaseSecret')")

    //Method find() was used to cover mrdr usecase when GatewayService is not fully provisioned in backup. Be aware that this parameter might be resolved to empty value on backup
    fixedParam("service-base.genAIURL", "((cmdbService.findBackingService('clusterguid=' + dynamicParams.ClusterGUID, 'type=GenAI Gateway Service Product').find('GenAIHubServiceBaseURL')?.trim()))?: 'NotConfigured'")
    fixedParam("service-base.genaiSmartChunkingURL", "cmdbService.find('GenAIAPIBaseURL', cmdbService.serviceNameFilter('GenAISmartChunking'))?.trim() ?: 'http://genai-api.genai-smart-chunking.svc.cluster.local:443'")

    // DB-TOOLS Environment variables
    fixedParam("enableDBTools", "dynamicParams.get('EnableDBTools')?.trim()")
    fixedParam("dbtools.image.tag", "'${dbtoolsVersion}'")
    fixedParam("dbtools.image.repository","""
       ({ ->
          def cpStage = cmdbService.labels().get('cpStageName')
          def repo = ''
          
          if (['development', 'integration', 'staging', 'trials'].contains(cpStage)) {
              repo = 'cirrus-docker.jfrog.io/platform-services/db-tools-service'
          } else if (cpStage == 'rnd-usgov' || cpStage == 'production-usgov') { 
              repo = 'artifactory-pcfrrd-cp-pl-endpoint.rnd-pcfrpegaservice.net:5000/platform-services/db-tools-service'
          } else { 
              repo = 'cirrus-docker-release.jfrog.io/platform-services/db-tools-service'
          } 
          return repo
       })()""".trimMargin())
    fixedParam("account","cmdbService.labels().get('accountid')")
    fixedParam("region", "cmdbService.cluster(dynamicParams.ClusterGUID).get('Region')")
    fixedParam("databaseID", "cmdbService.get('DatabaseID')")
    fixedParam("databaseName", "cmdbService.get('DatabaseName')")
    fixedParam("databaseHost", "cmdbService.get('DatabaseHost')")
    fixedParam("databasePort", "cmdbService.get('DatabasePort')")
    //Workaround for retrieving the DatabaseSecret in the ReadOnlyMode by reading it from ActiveRegion sce outputs - until US-641628 is completed on DBMS side.
    //fixedParam("databaseSecret", "(cmdbService.isDeploymentActive() ? cmdbService.get('DatabaseSecret') : (cmdbService.activeResource().get('DatabaseSecret')))")
    fixedParam("databaseSecret", "cmdbService.get('DatabaseSecret')")
    fixedParam("cloudProvider", "cmdbService.labels().get('provider')")

    output("ServiceEndpoint",
        "GenAI Vector Store URL",
        "'http://genai-vector-store.'.concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('.svc.cluster.local:443')")
    output("OpsServiceEndpoint",
        "GenAI Vector Store Ops Service URL",
        "'http://genai-vector-store-ops.'.concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('.svc.cluster.local:443')")
    output("OpsServiceEndpointExternal",
        "GenAI Vector Store Ops Service URL (External)",
        """
            (cmdbService.labels().get('provider') == 'gcp') ?
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('OperationsLoadBalancerDNSName')).concat('/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops') :
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('BackendLoadBalancerDNSName')).concat(':7443/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops')
        """.trimIndent())
    output("OpsApiSwaggerPath",
        "Ops Api Swagger Path for GOC troubleshooting framework",
        "'/swagger/ops.yaml'")
    output("OpsServiceNamespace",
        "Ops Service namespace",
        "dynamicParams.get('Namespace')?.trim()")
    output("OpsServiceName",
        "Ops Service namespace",
        "'genai-vector-store-ops'")
    output("OpsAuthType",
        "Ops Service namespace",
        "'SAX'")
    output("OpsBaseURL",
        "Ops Service Endpoint",
        """
            (cmdbService.labels().get('provider') == 'gcp') ?
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('OperationsLoadBalancerDNSName')).concat('/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops') :
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('BackendLoadBalancerDNSName')).concat(':7443/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops')
        """.trimIndent())

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "helm"
        template = "genai-vector-store-${project.version}.tgz" // packaged helm
        version = helmVersion.substringBeforeLast(".")
        timeout = "60m"
        skipUpdateDuringMRDRFailover = false
    } as groovy.lang.Closure<DeploymentRefConfig>)
}
