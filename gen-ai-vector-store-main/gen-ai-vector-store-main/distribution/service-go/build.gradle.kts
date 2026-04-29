/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */


plugins {
    id("com.pega.go")
    id("com.pega.contract")
}

go {
    downloadGo.set(true)
    downloadCuttyhunkCLI.set(false)
    forceSourcesZip.set(true)

    buildDir.set(File("${rootProject.projectDir}/cmd/service"))
    buildTarget.set(File("${rootProject.projectDir}/build/go/service"))
    testDir.set(File("${rootProject.projectDir}"))
    componentTestDir.set(File("${rootProject.projectDir}"))
//    integrationTestDir.set(File("${rootProject.projectDir}"))

    modules.add("${rootProject.projectDir}/cmd/service/...")
    modules.add("${rootProject.projectDir}/cmd/middleware")

    // internal modules are tested in the scope of the main service
    modules.add("${rootProject.projectDir}/internal/...")
}

// =============================================================================
// Pact Consumer Contract Tests
// =============================================================================
//
// gen-ai-vector-store is a CONSUMER of GenAI Gateway (gen-ai-hub-service).
// These tasks set up and run pact tests separately from component tests.

// Install pact-ruby-standalone for CI 
val pactInstallScript = "${rootProject.projectDir}/scripts/install-pact.sh"
val setupPact by tasks.registering {
    description = "Install pact-ruby-standalone for contract tests"
    doFirst {
        exec {
            commandLine("bash", "-c", "chmod +x ${pactInstallScript}")
        }
        exec {
            commandLine("bash", "-c", "${pactInstallScript} ${rootProject.projectDir}")
        }
    }
}

var isCI = System.getenv("CI") ?: "false"

// Dedicated pact test task - runs with "-tags pact" to compile tests with //go:build pact constraint
val goPactTests by tasks.registering(com.pega.gradle.plugins.go.tasks.GoTestTask::class) {
    description = "Run Pact consumer contract tests to generate contract files"
    group = "verification"

    dependsOn(setupPact)
    if (isCI == "true") { dependsOn(":configurePegaGitAccess") }

    // Configure for pact tests only - use SEPARATE output files to avoid Gradle task conflict
    xunitResultsFile.set(
        project.layout.buildDirectory.file("test-results/pactTest/TEST-GO.xml")
    )
    htmlReportDirectory.set(
        project.layout.buildDirectory.dir("reports/pactTest/gopacttests")
    )
    testSourceDirectory.set(rootProject.layout.projectDirectory)
    tags.set(setOf("pact"))

    // Ignoring tests failures to avoid breaking build if no pact tests are present
    // TODO: Consider adding check in doLast to fail if no pact files generated (non-critical)
    ignoreFailures.set(true)

    // Same environment as gocomponenttests to disable SAX checks
    environment.put("SAX_DISABLED", "true")
    environment.put("SAX_CLIENT_DISABLED", "true")
    environment.put("ISOLATION_ID_VERIFICATION_DISABLED", "true")

    mustRunAfter(
        ":distribution:background-go:downloadCuttyhunkCLI",
        ":distribution:ops-go:downloadCuttyhunkCLI",
        ":distribution:productcatalog:processResources",
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
        ":distribution:service-infrastructure-terraform:terraformExecExtract",
        ":distribution:service-infrastructure-terraform:getTFConfigFiles",
        ":distribution:service-infrastructure-terraform:terraformInit",
        ":distribution:background-go:goDistExtract",
        ":distribution:background-go:goDownload",
        ":distribution:ops-go:goDistExtract",
        ":distribution:ops-go:goDownload",
    )

    // Copy pact files to locations for pipeline pact publishing
    // Pact files are written to internal/embedders/pact/pact/ (relative to test package directory)
    // because Go tests run from their package directory, and PactDir: "pact" is relative
    doLast {
        val pactSourceDir = "${rootProject.projectDir}/internal/embedders/pact/pact"
        val pactBuildDir = "${project.layout.buildDirectory.get().asFile}/pacts"
        val contractTestDir = "${rootProject.projectDir}/src/contractTest/build/pacts"

        // Debug: List files in source directory
        val sourceDir = file(pactSourceDir)
        if (sourceDir.exists()) {
            val jsonFiles = sourceDir.listFiles()?.filter { it.name.endsWith(".json") } ?: emptyList()
            logger.lifecycle("Pact: Found ${jsonFiles.size} JSON files in $pactSourceDir")
            jsonFiles.forEach { logger.lifecycle("Pact:   - ${it.name}") }
        } else {
            logger.lifecycle("Pact: Source directory does not exist: $pactSourceDir")
        }

        // Create directories if they don't exist
        file(pactBuildDir).mkdirs()
        file(contractTestDir).mkdirs()

        // Copy to subproject build directory (for contract plugin - isConsumerContractAvailable checks here)
        copy {
            from(pactSourceDir)
            into(pactBuildDir)
            include("*.json")
        }

        // Copy to contractTest directory (for pipeline publishing)
        copy {
            from(pactSourceDir)
            into(contractTestDir)
            include("*.json")
        }

        // Create properties file for contract publishing
        val propertiesFile = file("${contractTestDir}/genaigateway.properties")
        propertiesFile.writeText("retention.state=DEVELOPMENT\n")

        // Verify files were copied
        val copiedFiles = file(pactBuildDir).listFiles()?.filter { it.name.endsWith(".json") } ?: emptyList()
        logger.lifecycle("Pact: Copied ${copiedFiles.size} JSON files to $pactBuildDir")
        copiedFiles.forEach { logger.lifecycle("Pact:   - ${it.name}") }
    }
}

// Configure gocomponenttests - runs standard component tests ONLY (not pact tests).
// Pact tests require -tags=pact build flag and are executed by goPactTests task.
// setupPact dependency ensures pact binaries are available for both tasks.
tasks.gocomponenttests.configure {
    dependsOn(setupPact)
    if (isCI == "true") { dependsOn(":configurePegaGitAccess") }

    ignoreFailures.set(true)

    environment.put("SAX_DISABLED", "true")
    environment.put("SAX_CLIENT_DISABLED", "true")
    environment.put("ISOLATION_ID_VERIFICATION_DISABLED", "true")

    mustRunAfter(
        ":distribution:background-go:downloadCuttyhunkCLI",
        ":distribution:ops-go:downloadCuttyhunkCLI",
        ":distribution:productcatalog:processResources",
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
        ":distribution:service-infrastructure-terraform:terraformExecExtract",
        ":distribution:service-infrastructure-terraform:getTFConfigFiles",
        ":distribution:service-infrastructure-terraform:terraformInit",
        ":distribution:background-go:goDistExtract",
        ":distribution:background-go:goDownload",
        ":distribution:ops-go:goDistExtract",
        ":distribution:ops-go:goDownload",
    )

}

// Make componentTest lifecycle task depend on goPactTests (using afterEvaluate because componentTest 
// is created by the contract plugin during configuration phase)
// This ensures pact files are generated BEFORE publishPacts checks for them
afterEvaluate {
    tasks.findByName("componentTest")?.dependsOn(goPactTests)
}

tasks.gobuild.configure {
    environment.put("CGO_ENABLED", "0")
    sourceDirectory.set(File("${rootProject.projectDir}/cmd/service"))
}

// Disable modifyPathsInCoverageTask - this task rewrites paths assuming sources in src/main/go/
// but this project has Go sources in the repository root. Same pattern is used in other Pega projects
tasks.matching { it.name.startsWith("modifyPathsInCoverageTask") }.configureEach {
    enabled = false
}

tasks.gotests.configure {
    environment.put("SAX_DISABLED", "true")
    environment.put("SAX_CLIENT_DISABLED", "true")
    environment.put("ISOLATION_ID_VERIFICATION_DISABLED", "true")

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

        ":distribution:ops-go:gotests",
        ":distribution:background-go:gotests",

        )

    if (isCI == "true") { dependsOn(":configurePegaGitAccess") }
}