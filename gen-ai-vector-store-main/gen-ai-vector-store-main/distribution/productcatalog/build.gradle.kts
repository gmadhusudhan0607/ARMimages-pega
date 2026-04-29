/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
 
import org.apache.tools.ant.filters.ReplaceTokens

plugins {
    id("com.pega.java")
}

val pegaServiceId: String by project
val pegaServiceName: String by project
val serviceAuthenticationClientServiceVersion: String by project
val dbInstanceVersion: String by project
val infinityIsolationVersion: String by project

tasks.named<Copy>("processResources") {
    inputs.property("version", version.toString())
    inputs.property("pegaServiceId", pegaServiceId.toString())
    inputs.property("pegaServiceName", pegaServiceName.toString())
    inputs.property("serviceAuthenticationClientServiceVersion", serviceAuthenticationClientServiceVersion.toString())
    inputs.property("dbInstanceVersion", dbInstanceVersion.toString())
    inputs.property("infinityIsolationVersion", infinityIsolationVersion.toString())
    // Gradle expand didn't work as yml contains !fn expressions, had to use replace token
    filter<ReplaceTokens>("tokens" to mapOf(
        "version" to version.toString(),
        "pegaServiceId" to pegaServiceId.toString(),
        "pegaServiceName" to pegaServiceName.toString(),
        "serviceAuthenticationClientServiceVersion" to serviceAuthenticationClientServiceVersion.toString(),
        "dbInstanceVersion" to dbInstanceVersion.toString(),
        "infinityIsolationVersion" to infinityIsolationVersion.toString()
    ))
}

publishing {
    publications {
        named<MavenPublication>("mavenDefault") {
            groupId = "com.pega.cloud.productcatalog"
            artifactId = "genai-vector-store"
        }
    }
}

security { enabled.set(false) }
