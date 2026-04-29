import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild


plugins {
    id("com.pega.quality")
    id("com.pega.release.cibuildversion")
    id("com.pega.veracode")
    id("com.pega.golang")
    id("com.pega.cloud.services.changelog")
    //Add plugin but do not APPLY.
    //This line ensures that there are no
    //ClassCastExceptions when referencing tasks from the distribution/docker
    //sub-project from other sub-projects (like distribution/helm)
    id("com.pega.dockercli.image") apply false
    id("com.pega.helmcli.helm") apply false
}
changelog {
    serviceOwnerEmail.set("Jarvis@pega.com")
    defaultBranch.set("main")
}

sonarqube {
    properties {
        property("sonar.projectVersion", version)

        property("sonar.sources", "cmd,internal")
        property("sonar.exclusions", "**/*_test.go,**/mock_*")

        property("sonar.tests", "test")
        property("sonar.test.inclusions", "**/*_test.go")

        property("sonar.go.coverage.reportPaths", "coverage.out")
    }

}

description = "genai-hub-service"

golang {
    version.set("1.25.8")
}

// Define bitbucketUser and bitbucketPassword in  ~/.gradle/gradle.properties
// You can use  HTTP token instead of Password,
// HTTP token can be generated on git.pega.com (Under "Manage Account" -> "HTTP access tokens")
val bitbucketUser: String by project
val bitbucketPassword: String by project

var bitbucketUsr: String = System.getenv("BITBUCKET_USR") ?: bitbucketUser
var bitbucketPwd: String = System.getenv("BITBUCKET_PSW") ?: bitbucketPassword

var isCI = System.getenv("CI") ?: "false"

//TODO:adjust once BUG-849378 is resolved
//workaround
tasks.makeTest.configure{

    if (isCI == "true") { dependsOn("configurePegaGitAccess") }

    dependsOn(":distribution:sax-registration-sce:processResources")
    dependsOn(":distribution:sax-registration-terraform:terraformExecExtract")
    dependsOn(":distribution:sax-registration-terraform:cleanTFWorkingDirectory")
    dependsOn(":distribution:sax-registration-terraform:getTFConfigFiles")
    dependsOn(":distribution:sax-registration-terraform:terraformInit")
    dependsOn(":distribution:sax-registration-terraform:distZip")

    dependsOn(":distribution:role-sce:processResources")
    dependsOn(":distribution:role-terraform:terraformExecExtract")
    dependsOn(":distribution:role-terraform:getTFConfigFiles")
    dependsOn(":distribution:role-terraform:getTFDependencies")
    dependsOn(":distribution:role-terraform:terraformInit")
    dependsOn(":distribution:role-terraform:distZip")

    dependsOn(":distribution:genai-private-model-config-sce:processResources")
    dependsOn(":distribution:genai-private-model-config-terraform:terraformExecExtract")
    dependsOn(":distribution:genai-private-model-config-terraform:getTFConfigFiles")
    dependsOn(":distribution:genai-private-model-config-terraform:getTFDependencies")
    dependsOn(":distribution:genai-private-model-config-terraform:terraformInit")
    dependsOn(":distribution:genai-private-model-config-terraform:distZip")

    dependsOn(":distribution:genai-private-model-externalsecret-sce:processTestResources")
    dependsOn(":distribution:genai-private-model-externalsecret-sce:processResources")
    dependsOn(":distribution:genai-private-model-externalsecret-helm:helmPackageMain")
    dependsOn(":distribution:genai-private-model-externalsecret-helm:helmPackageFilterMain")
    dependsOn(":distribution:genai-private-model-externalsecret-helm:helmLintMain")

    dependsOn(":distribution:genai-hub-service-sce:processTestResources")
    dependsOn(":distribution:genai-hub-service-sce:processResources")
    dependsOn(":distribution:genai-hub-service-helm:helmPackageMain")
    dependsOn(":distribution:genai-hub-service-helm:helmPackageFilterMain")
    dependsOn(":distribution:genai-hub-service-helm:helmLintMain")

    dependsOn(":distribution:sax-iam-oidc-provider-sce:processResources")
    dependsOn(":distribution:sax-iam-oidc-provider-terraform:getTFConfigFiles")
    dependsOn(":distribution:sax-iam-oidc-provider-terraform:getTFDependencies")
    dependsOn(":distribution:sax-iam-oidc-provider-terraform:terraformInit")
    dependsOn(":distribution:sax-iam-oidc-provider-terraform:distZip")

    dependsOn(":distribution:genai-awsbedrock-infra-sce:processResources")
    dependsOn(":distribution:genai-awsbedrock-infra-terraform:getTFConfigFiles")
    dependsOn(":distribution:genai-awsbedrock-infra-terraform:getTFDependencies")
    dependsOn(":distribution:genai-awsbedrock-infra-terraform:terraformInit")
    dependsOn(":distribution:genai-awsbedrock-infra-terraform:distZip")

    // Add GCP Vertex infra projects
    dependsOn(":distribution:genai-gcpvertex-host-sce:processResources")
    dependsOn(":distribution:genai-gcpvertex-host-terraform:getTFConfigFiles")
    dependsOn(":distribution:genai-gcpvertex-host-terraform:getTFDependencies")
    dependsOn(":distribution:genai-gcpvertex-host-terraform:terraformInit")
    dependsOn(":distribution:genai-gcpvertex-host-terraform:distZip")

    dependsOn(":distribution:genai-gcpvertex-infra-sce:processResources")
    dependsOn(":distribution:genai-gcpvertex-infra-terraform:getTFConfigFiles")
    dependsOn(":distribution:genai-gcpvertex-infra-terraform:getTFDependencies")
    dependsOn(":distribution:genai-gcpvertex-infra-terraform:terraformInit")
    dependsOn(":distribution:genai-gcpvertex-infra-terraform:distZip")

    dependsOn(":distribution:genai-defaults-sce:processResources")
    dependsOn(":distribution:genai-defaults-terraform:getTFConfigFiles")
    dependsOn(":distribution:genai-defaults-terraform:getTFDependencies")
    dependsOn(":distribution:genai-defaults-terraform:terraformInit")
    dependsOn(":distribution:genai-defaults-terraform:distZip")

    dependsOn(":distribution:product-catalog:processResources")
}


var userHomeDir = System.getenv("HOME") ?: "."

tasks.register("integrationTest") {
    // Test service integration locally with mocked models
    if (isCI == "true") { dependsOn("configurePegaGitAccess") }
    dependsOn("integrationTestInfrastructureUp")
    dependsOn("integrationTestRun")

    // Test SCE
    if (isCI == "true") {
        dependsOn(":distribution:genai-hub-service-sce:validateOutputs")
        finalizedBy(":distribution:genai-hub-service-sce:cleanUpResources")
    }
}

// Configure GIT to be able to get pkgs from git.pega.io and github.com
tasks.register("configurePegaGitAccess") {
    val githubUsr: String = System.getenv("GITHUB_USR") ?: bitbucketUsr
    val githubPwd: String = System.getenv("GITHUB_PSW") ?: bitbucketPwd
    val pegaCredLine = "machine git.pega.io login $bitbucketUsr password $bitbucketPwd"
    val githubCredLine = "machine github.com login $githubUsr password $githubPwd"
    val netrcFile = file("$userHomeDir/.netrc")
    for (credLine in listOf(pegaCredLine, githubCredLine)) {
        if (!netrcFile.exists()) {
            netrcFile.appendText("$credLine\n")
        } else {
            if (!netrcFile.readLines().contains(credLine)) {
                netrcFile.appendText("$credLine\n")
            }
        }
    }
}

tasks.register<Exec>("integrationTestInfrastructureUp") {
    dependsOn(":distribution:genai-hub-service-docker:buildImage")
    dependsOn(":distribution:genai-gateway-ops-docker:buildImage")
    val dkBuildImage = project(":distribution:genai-hub-service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBuildOpsImage = project(":distribution:genai-gateway-ops-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("SERVICE_DOCKER_OPS_IMAGE", dkBuildOpsImage.get().image.get())
    }
    commandLine("make", "integration-test-up")
}

tasks.register<Exec>("integrationTestInfrastructureDown") {
    val dkBuildImage = project(":distribution:genai-hub-service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBuildOpsImage = project(":distribution:genai-gateway-ops-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("SERVICE_DOCKER_OPS_IMAGE", dkBuildOpsImage.get().image.get())
    }
    commandLine("make", "integration-test-down")
}

tasks.register<Exec>("integrationTestRun") {
    mustRunAfter("integrationTestInfrastructureUp")
    finalizedBy("integrationTestInfrastructureDown")
    val dkBuildImage = project(":distribution:genai-hub-service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBuildOpsImage = project(":distribution:genai-gateway-ops-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("SERVICE_DOCKER_OPS_IMAGE", dkBuildOpsImage.get().image.get())
    }
    commandLine("make", "integration-test-run")
}

tasks.register<Delete>("cleanBin"){
    delete(file("bin"))
}

tasks.named("clean"){
    dependsOn("cleanBin")
}

tasks.register("integrationTestWithoutCleanup") {
    // Test service integration locally with mocked models
    if (isCI == "true") { dependsOn("configurePegaGitAccess") }
    dependsOn("integrationTestInfrastructureUp")
    dependsOn("integrationTestRunWithoutCleanup")

    // Test SCE
    if (isCI == "true") {
        dependsOn(":distribution:genai-hub-service-sce:validateOutputs")
        finalizedBy(":distribution:genai-hub-service-sce:cleanUpResources")
    }
}

tasks.register<Exec>("integrationTestRunWithoutCleanup") {
    mustRunAfter("integrationTestInfrastructureUp")
    val dkBuildImage = project(":distribution:genai-hub-service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBuildOpsImage = project(":distribution:genai-gateway-ops-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("SERVICE_DOCKER_OPS_IMAGE", dkBuildOpsImage.get().image.get())
    }
    commandLine("make", "integration-test-run")
}