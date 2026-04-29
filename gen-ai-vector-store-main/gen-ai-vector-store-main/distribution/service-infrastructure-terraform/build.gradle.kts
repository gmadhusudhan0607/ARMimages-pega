/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

plugins {
    id("com.pega.terraform")
}

val pegasecProviderVersion: String by rootProject

terraform {
    variables.put("namespace", "${rootProject.name}-tf")
    variables.put("pegasec_version", pegasecProviderVersion)
}

distributions {
    main {
        contents {
            filesMatching("**/versions.tf") {
                expand(
                        "var" to mapOf(
                                "pegasec_version" to pegasecProviderVersion
                        )
                )
            }
        }
    }
}

dependencies {
    terraform("com.pega.cloud.services:sax-tf-provider:${pegasecProviderVersion}@zip")
}
