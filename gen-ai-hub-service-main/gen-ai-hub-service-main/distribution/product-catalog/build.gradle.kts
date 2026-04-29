/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

import org.apache.tools.ant.filters.ReplaceTokens

plugins {
    id("com.pega.quality")
    id("com.pega.java")
}

val serviceAuthenticationClientServiceVersion: String by project


tasks.named<Copy>("processResources") {
    inputs.property("serviceAuthenticationClientServiceVersion", serviceAuthenticationClientServiceVersion.toString())
    inputs.property("serviceVersionNumber", version.toString())
    filter<ReplaceTokens>("tokens" to mapOf(
        "serviceAuthenticationClientServiceVersion" to serviceAuthenticationClientServiceVersion.toString(),
        "serviceVersionNumber" to version.toString(),
        "version" to version.toString()
    ))
}

tasks.test {
    val artifactoryURL: String by project
    val artifactoryUser: String by project
    val artifactoryPassword: String by project
    val artifactoryVirtualRepo: String by project
    useJUnit()
    systemProperty("artifactoryURL", artifactoryURL)
    systemProperty("artifactoryUser", artifactoryUser)
    systemProperty("artifactoryPassword", artifactoryPassword)
    systemProperty("artifactRepository", artifactoryVirtualRepo)
}

publishing {
    publications {
        named<MavenPublication>("mavenDefault") {
            groupId = "com.pega.cloud.productcatalog"
            artifactId = "genai-gateway-service-product"
        }
    }
} 
