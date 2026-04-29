/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

plugins {
    id("com.pega.go")
}

go {
    downloadGo.set(true)
    downloadCuttyhunkCLI.set(false)
    forceSourcesZip.set(true)

    buildDir.set(File("${rootProject.projectDir}/cmd/background"))
    buildTarget.set(File("${rootProject.projectDir}/build/go/background"))
    testDir.set(File("${rootProject.projectDir}/cmd/background"))
//    componentTestDir.set(File("${rootProject.projectDir}/cmd/background"))
//    integrationTestDir.set(File("${rootProject.projectDir}/cmd/background"))

    modules.add("${rootProject.projectDir}/cmd/background/...")
//    modules.add("${rootProject.projectDir}/cmd/middleware")
}

tasks.gobuild.configure {
    environment.put("CGO_ENABLED", "0")
    sourceDirectory.set(File("${rootProject.projectDir}/cmd/background"))
}

var isCI = System.getenv("CI") ?: "false"

tasks.gotests.configure {
    environment.put("SAX_DISABLED", "true")
    environment.put("SAX_CLIENT_DISABLED", "true")
    mustRunAfter(
        ":distribution:productcatalog:jacocoTestReport",
        "::distribution:productcatalog:processResources",
        ":distribution:role-sce:processResources",
        ":distribution:isolation-sce:processResources",
        ":distribution:sax-registration-sce:processResources",
        ":distribution:service-sce:processResources",
        ":distribution:service-infrastructure-sce:processResources",
        ":distribution:isolation-terraform:terraformExecExtract",
        ":distribution:isolation-terraform:getTFConfigFiles",
        ":distribution:isolation-terraform:getTFDependencies",
        ":distribution:isolation-terraform:terraformInit",
        ":distribution:role-terraform:terraformExecExtract",
        ":distribution:role-terraform:getTFConfigFiles",
        ":distribution:role-terraform:getTFDependencies",
        ":distribution:role-terraform:terraformInit",
        ":distribution:sax-registration-terraform:terraformExecExtract",
        ":distribution:sax-registration-terraform:getTFConfigFiles",
        ":distribution:sax-registration-terraform:getTFDependencies",
        ":distribution:sax-registration-terraform:terraformInit",
        ":distribution:service-helm:helmPackageFilterMain",
        ":distribution:service-infrastructure-terraform:terraformExecExtract",
        ":distribution:service-infrastructure-terraform:getTFConfigFiles",
        ":distribution:service-infrastructure-terraform:terraformInit",
    )
    if (isCI == "true") { dependsOn(":configurePegaGitAccess") }
}