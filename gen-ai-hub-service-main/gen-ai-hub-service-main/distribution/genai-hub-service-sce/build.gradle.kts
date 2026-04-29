import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

import com.pega.gradle.plugin.tasks.RetrieveServiceOutputTask
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import java.util.UUID

group = "com.pega.provisioning.services"
description = "Deployment template to provision genai-hub-service in k8s cluster"

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
    id("com.pega.helmcli.helm")
}

val serviceAuthenticationClientServiceVersion: String by project

// List dependencies. Can have assets, services, optionalServices, testCompile
dependencies {
    assets(project(":distribution:genai-hub-service-helm", "archives"))
    // adding dependency on ServiceAuthenticationClientService, since we need its output to be mounted in the genai-ops pod
    services(group="com.pega.provisioning.services", name="ServiceAuthenticationClientService", version=serviceAuthenticationClientServiceVersion)
    optionalServices(project(":distribution:role-sce"))
}

evaluationDependsOn(":distribution:genai-hub-service-docker")
evaluationDependsOn(":distribution:genai-gateway-ops-docker")

val serviceNamespace: String by project
val helmVersion: String by project
val testNamespace = "${serviceNamespace}-test-" + generateLet()

val dockerBuildImage = project(":distribution:genai-hub-service-docker").tasks.named<DockerImageBuild>("buildImage")
val dockerRepo = dockerBuildImage.get().image.name.orNull
val dockerBuildOpsImage = project(":distribution:genai-gateway-ops-docker").tasks.named<DockerImageBuild>("buildImage")
val dockerOpsRepo = dockerBuildOpsImage.get().image.name.orNull

ext["testNamespace"] = testNamespace

val outGenAIHubServiceBaseURL = "http://genai-hub-service.${testNamespace}.svc.cluster.local:443"
val outTargetGenAIURL: String by project
val outTargetSelfStudyBuddyURLv1: String by project
ext["outGenAIHubServiceBaseURL"] = outGenAIHubServiceBaseURL
ext["outTargetGenAIURL"] = outTargetGenAIURL
ext["outTargetSelfStudyBuddyURLv1"] = outTargetSelfStudyBuddyURLv1

sar {
    name = "GenAIHubService"
    description = "GenAI Hub Service"

    dynamicParam("ProvisioningType", "Determines how many prompts are shown. Standard option will show fewer prompts compared to Advanced. Default = Standard")
    dynamicParam("ClusterGUID", "GUID of CloudK cluster where service is deployed")
    dynamicParam("GenAIURL", "(Optional) The URL of GenAI API")
    dynamicParam("DockerRegistry","The Docker Registry for the service images")
    dynamicParam("Namespace", "K8s namespace in which the service is created")
    dynamicParam("SelfStudyBuddyURLv1", "(Optional) The URL of SelfStudy (Knowledge) Buddy API v1")
    dynamicParam("DemoAwsBedrockURL","(Optional) The URL of AWS Bedrock for demonstration purpose. Not present for Production environment.")
    dynamicParam("DemoGcpVertexURL","(Optional) The URL of GCP Vertex AI for demonstration purpose. Not present for Production environment.")
    dynamicParam("UseVertexAIInfra", "GCP VertexAI Endpoint provisioned via GenAI Infrastructure for GCP")
    dynamicParam("PlatformType", "The type of platform that is deployed in the Cluster i.e. Infinity or Launchpad")
    dynamicParam("UseSax", "Issue SAX tokens")
    dynamicParam("LLMAccountID", "AWS Account ID where the AWS Bedrock LLMs are hosted")
    dynamicParam("LLMModelsRegion", "The AWS Region where the AWS Bedrock LLMs are hosted")
    dynamicParam("UseOtlp", "Decides whether to use OTLP or not.")
    dynamicParam("OtlpSampler","Opentelemetry sampler (Defaults to parentbased_traceidratio)")
    dynamicParam("OtlpSamplerArg","Sampler argument value, ranges from 0 to 1 (Defaults to 0.1, means 10%)")

    dynamicParam("CopyrightProtection","[Experimental] Avoid Copyright Infringements")
    dynamicParam("MaxTokensStrategy","Max Output Tokens Adjustment Strategy")
    dynamicParam("MaxTokensBaseValue","Max Output Tokens Base Value")
    dynamicParam("MaxTokensForced","Force Max Output Tokens Adjustment even if se")
    dynamicParam("MaxTokensInStreaming","Force Max Output Tokens Adjustment in streaming calls")
    dynamicParam("ProcessingCacheSize","Request Cache Size (samples) form max-tokens adjustment")

    dynamicParam("RequestedCpu", "CPU requested for the service container")
    dynamicParam("RequestedMemory", "Memory requested for the service container")
    dynamicParam("LimitCpu", "CPU limit for the service container")
    dynamicParam("LimitMemory", "Memory limit for the service container")
    dynamicParam("LogLevel","(Optional) Log level")

    dynamicParam("UseGenAIInfra", "Flag to enable the use of models provisioned via GenAI Infra Contorl Plane")
    dynamicParam("UseAutoMapping", "Flag to enable the use of AutoMapping via Ops Service mapping fetching")
    dynamicParam("SaxCell", "SAX Cell for OIDC calls - US, EU or APAC")
    dynamicParam("DisplayPreviewModels","Displays models with preview lifecycle on list models endpoint")
    dynamicParam("SmartModelOverride", "(Optional) Override for the default smart model. Takes precedence over models defined by GenAI Infrastructure.")
    dynamicParam("FastModelOverride", "(Optional) Override for the default fast model. Takes precedence over models defined by GenAI Infrastructure.")
    dynamicParam("ProModelOverride", "(Optional) Override for the default pro model. Takes precedence over models defined by GenAI Infrastructure.")
    dynamicParam("EnabledProviders", "The enabled providers for the GenAI gateway service.")
    dynamicParam("EnableProModelDefault", "Enable Pro tier in default model responses. When false, maintains backward compatibility by excluding Pro field from API responses.")
    dynamicParam("ModelTimeoutSeconds", "Client timeout in seconds that Gateway will wait for a model call to complete before close connection.")

    fixedParam("kubeconfig", "cmdbService.cluster(dynamicParams.ClusterGUID).get('kubeconfig')")
    fixedParam("AccountID","cmdbService.cluster(dynamicParams.ClusterGUID).get('AccountID')")
    fixedParam("namespace","dynamicParams.Namespace ?: cmdbService.find('Namespace')")
    fixedParam("service-base.ClusterGUID","dynamicParams.ClusterGUID")
    fixedParam("saxClientDetailsArn", "cmdbService.labels().find('provider') == 'aws' ? cmdbService.get('ServAuthJSONSecretARN') : cmdbService.get('ServAuthJSONSecretARN').split('/').last()")
    fixedParam("service-base.ClusterName","cmdbService.cluster(dynamicParams.ClusterGUID).get('ClusterID')") // CMDB.find expression is returning null value for ClusterName
    fixedParam("service-base.pegaservices.genai-hub-service.image.tag","'${project.version}'")
    fixedParam("service-base.pegaservices.genai-hub-service.image.repository","dynamicParams.DockerRegistry.concat('/').concat('${dockerRepo}')")

    fixedParam("service-base.pegaservices.genai-hub-service.container.resources.requests.cpu","dynamicParams.RequestedCpu")
    fixedParam("service-base.pegaservices.genai-hub-service.container.resources.requests.memory","dynamicParams.RequestedMemory")
    fixedParam("service-base.pegaservices.genai-hub-service.container.resources.limits.cpu","dynamicParams.LimitCpu")
    fixedParam("service-base.pegaservices.genai-hub-service.container.resources.limits.memory","dynamicParams.LimitMemory")

    fixedParam("service-base.pegaservices.genai-gateway-ops.image.tag","'${project.version}'")
    fixedParam("service-base.pegaservices.genai-gateway-ops.image.repository","dynamicParams.DockerRegistry.concat('/').concat('${dockerOpsRepo}')")

    fixedParam("service-base.genAIURL", "dynamicParams.get('GenAIURL')?.trim()")
    fixedParam("service-base.demoGcpVertexURL", "dynamicParams.get('DemoGcpVertexURL')?.trim()")
    fixedParam("service-base.useVertexAIInfra", "dynamicParams.get('UseVertexAIInfra')")
    fixedParam("service-base.logLevel", "dynamicParams.get('LogLevel')?.trim()")
    fixedParam("service-base.useGenAIInfra", "dynamicParams.UseGenAIInfra")
    fixedParam("service-base.useAutoMapping", "dynamicParams.UseAutoMapping")
    fixedParam("service-base.stageName", "cp.getStageName()")
    fixedParam("service-base.saxCell", "dynamicParams.get('SaxCell')?.trim()")
    fixedParam("service-base.llmAccountID","dynamicParams.get('LLMAccountID')?.trim()")
    fixedParam("service-base.llmModelsRegion","dynamicParams.get('LLMModelsRegion')?.trim()")
    fixedParam("service-base.displayPreviewModels", "dynamicParams.get('DisplayPreviewModels')")
    fixedParam("service-base.smartModelOverride", "dynamicParams.get('SmartModelOverride')?.trim()")
    fixedParam("service-base.fastModelOverride", "dynamicParams.get('FastModelOverride')?.trim()")
    fixedParam("service-base.proModelOverride", "dynamicParams.get('ProModelOverride')?.trim()")
    fixedParam("service-base.enabledProviders", "dynamicParams.EnabledProviders")
    fixedParam("service-base.modelTimeoutSeconds", "dynamicParams.ModelTimeoutSeconds")
    fixedParam("service-base.enableProModelDefault", "dynamicParams.EnableProModelDefault")


    // Standardized parameter names for target URLs for different providers
    fixedParam("service-base.targetUrlAzureOpenai", "dynamicParams.get('GenAIURL')?.trim()")
    fixedParam("service-base.targetUrlAwsBedrock", "dynamicParams.get('DemoAwsBedrockURL')?.trim()")
    fixedParam("service-base.targetUrlGcpVertex", "dynamicParams.get('DemoGcpVertexURL')?.trim()")
    fixedParam("service-base.requestProcessingCopyrightProtection", "dynamicParams.get('CopyrightProtection')?.trim()")
    fixedParam("service-base.requestProcessingOutputTokensAdjustmentStrategy", "dynamicParams.get('MaxTokensStrategy')?.trim()")
    fixedParam("service-base.requestProcessingOutputTokensBaseValue", "dynamicParams.get('MaxTokensBaseValue')?.trim()")
    fixedParam("service-base.requestProcessingOutputTokensAdjustmentForced", "dynamicParams.get('MaxTokensForced')?.trim()")
    fixedParam("service-base.requestProcessingOutputTokensAdjustmentInStreaming", "dynamicParams.get('MaxTokensInStreaming')?.trim()")
    fixedParam("service-base.requestProcessingCacheSize", "dynamicParams.get('ProcessingCacheSize')?.trim()")

    // OpenTelemetry Endpoint and Sampling configuration, it should be used when OpenTelemetry tracing library is used
    // Default value for sampler argument is 0.1, it can be overridden as following in service
    fixedParam("service-base.OtlpEndpoint","cmdbService.cluster(dynamicParams.ClusterGUID).find('OtlpEndpoint')")
    fixedParam("service-base.OtlpSampler","dynamicParams.OtlpSampler")
    fixedParam("service-base.OtlpSamplerArg","dynamicParams.OtlpSamplerArg")
    fixedParam("service-base.SamplingPercentage","dynamicParams.OtlpSamplerArg != null ? String.valueOf(Double.parseDouble(dynamicParams.OtlpSamplerArg) * 100) : String.valueOf(0.1 * 100)")

    // Flag for Launchpad UAS authentication
    fixedParam("service-base.platformType", "dynamicParams.get('PlatformType')")

    // Flag to enable GenAI Hub Service to issue SAX tokens to access model deployments
    fixedParam("service-base.useSax", "dynamicParams.get('UseSax')")

    fixedParam("StageName", "cp.getStageName()")

    fixedParam("service-base.region",
        "dynamicParams.get('UseSax')?.trim().equals('true') ? cmdbService.cluster(dynamicParams.ClusterGUID).get('Region') : ''")
    fixedParam("cloudProvider", "cmdbService.labels().find('provider') ?: 'aws'")

    // Input to assign IAM Role to the pod to access SAX Secret
    fixedParam("service-base.pegaservices.genai-hub-service.serviceIRSARole",
        "dynamicParams.get('UseSax')?.trim().equals('true') ? cmdbService.find('ServiceAccountRole') : ''")

    output("GenAIHubServiceBaseURL", "GenAI Hub Service URL", "'http://genai-hub-service.'.concat(dynamicParams.Namespace ?: cmdbService.find('Namespace')).concat('.svc.cluster.local:443')")

    // Those parameters are used by integration test to validate that we can resolve URL from cp-setting maps
    output("TargetGenAIURL", "Target URL of GenAI API", "dynamicParams.GenAIURL")
    output("TargetSelfStudyBuddyURLv1", "Target Knowledge Buddy API", "dynamicParams.SelfStudyBuddyURLv1")

    output("OpsBaseURL",
        "Ops Service Endpoint through the Cluster LoadBalancer",
        """
            (cmdbService.labels().get('provider') == 'gcp') ?
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('OperationsLoadBalancerDNSName')).concat('/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops') :
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('BackendLoadBalancerDNSName')).concat(':7443/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops')
        """.trimIndent())

    output("OpsServiceEndpoint",
        "GenAI Gateway Ops Internal Cluster Service URL",
        "'http://genai-gateway-ops.'.concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('.svc.cluster.local:443')")

    output("OpsServiceEndpointExternal",
        "GenAI Gateway Ops Service URL (External)",
        """
            (cmdbService.labels().get('provider') == 'gcp') ?
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('OperationsLoadBalancerDNSName')).concat('/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops') :
            'https://'.concat(cmdbService.cluster(dynamicParams.ClusterGUID).get('BackendLoadBalancerDNSName')).concat(':7443/ops/').concat(dynamicParams.Namespace ?: '${serviceNamespace}').concat('-ops')
        """.trimIndent())

    output("OpsServiceNamespace","Ops Service namespace","dynamicParams.get('Namespace')?.trim()")
    output("OpsServiceName","Ops Service name","'genai-gateway-ops'")
    output("OpsAuthType","Ops Service auth method","")

    output("ServiceNamespace","Service namespace","dynamicParams.get('Namespace')?.trim()")
    output("ServiceName","Ops Service name","'genai-hub-service'")
    output("SaxCell", "SaxCell aligned with this provisionig", "dynamicParams.get('SaxCell')")
    output("GenAIDefaultModelsEndpoint", "GenAI Default Fasta and Smart Models Endpoint", "'http://genai-hub-service.'.concat(dynamicParams.Namespace ?: cmdbService.find('Namespace')).concat('.svc.cluster.local:443/models/defaults')")

    deploymentRef("helm",
        configurations.assets.get().files.first().name, // resource, which contains the packaged helm
        "3.18", //Helm version
        "10m"  //Helm timeout
    )
}

tasks.register<Copy> ("generateDefaults") {
    from ("src/test/resources/templates/ps-defaults.json")
    from ("src/test/resources/templates/ps-createServiceInput.json")
    into ("build/provisioning/generated/")
}

val backingServiceGUID by extra(UUID.randomUUID().toString())
val uniqueNamespaceForTest:String by extra.properties

//The extension 'provisioningService' intakes the properties needed to invoke the managed control-plane services,
// Also injects the properties required to deploy the current SCE, like resourceType, resourceGUID, serviceNamespace.
provisioningService {
    // Mandatory parameters
    clientName = "024877532571" // Pega internal AWS account
    awsProfile = "default"
    stage = "integration"
    resourceType = "backing-service" // The resource-type for deploying the SCE being tested
    psDefaultsTask = project.tasks.findByName("generateDefaults")

    // Optional parameters
    resourceGUID = "@@backingServiceGUID" // The resource-guid for deploying the SCE being tested. If not provided, plugin will generate one.
}

fun readFileAsTextUsingInputStream(fileName: String)
        = File(fileName).inputStream().readBytes().toString(Charsets.UTF_8)

tasks.register<RetrieveServiceOutputTask>("validateOutputs") {
    dependsOn("createService")
    fileName = "output.json"
    resourceType = "backing-service"
    resourceGUID = "@@backingServiceGUID"
    serviceName = sar.name

    doLast {
        var serviceEndpointURL = ""
        var data = File(project.buildDir.absolutePath + "/" + fileName).readText(Charsets.UTF_8)
        println("service output :- $data")
        val outputs: List<Map<String, String>> = Gson().fromJson(data, object : TypeToken<List<Map<String, String>>>() {}.type)
        // use below flag to address conditional checks
        var foundURL = false
        outputs.forEach {
            if("GenAIHubServiceBaseURL" == it["name"]!!) {
                foundURL = true
                serviceEndpointURL = it["value"]!!
            }
        }
        if(!foundURL) { throw GradleException("Could not find output GenAIHubServiceBaseURL")}
        project.ext["serviceEndpointURL"] = serviceEndpointURL
    }
}

tasks.register("cleanUpResources") {
    dependsOn("deleteResource")
    finalizedBy("deleteTestSCEs")
}

fun generateLet(): String {
    val characters = ('a'..'z') + ('0'..'9')
    return (1..5)
            .map { characters.random() }
            .joinToString("")
}
