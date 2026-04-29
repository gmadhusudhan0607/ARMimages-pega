/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

import org.gradle.internal.os.OperatingSystem

plugins {
    id("com.pega.terraform")
}

val psRestApiProviderVersion: String by rootProject
var psRestApiProviderVer = psRestApiProviderVersion

if (OperatingSystem.current().isMacOsX) {
    psRestApiProviderVer = "$psRestApiProviderVersion-darwin"
}

terraform {
    variables.put("namespace", "${rootProject.name}-tf")
}

dependencies {
    terraform("com.pega.cloud.services.provisioning-service:ps-restapi-tf-provider:${psRestApiProviderVer}@zip")
}

tasks.register<Exec>("terraform-lock-file-cleanup") {
    commandLine("make", "terraform-lock-clean")
    mustRunAfter("terraformInit")
}

tasks.distZip.configure{
    dependsOn("terraform-lock-file-cleanup")
}