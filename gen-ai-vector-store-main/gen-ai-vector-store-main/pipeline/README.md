## Pipeline Configuration Capabilities

Properties that may be defined in the file and used by the pipeline include:

| Property | Description | Default |
| -------- | ----------- | ------- |
| `COMPONENT_TEST_NODE` | optionally configure the jenkins executor node for the component test stage to be executed on | The default node used for the rest of the build |
| `COMPONENT_TEST_TIMEOUT` | optionally configure the timeout value (in minutes) for the component test stage of the pipeline | 120 |
| `MICRO_BENCHMARK_TEST_NODE` | optionally configure the jenkins executor node for the micro benchmark test stage to be executed on | The default node used for the rest of the build |
| `MICRO_BENCHMARK_TEST_TIMEOUT` | optionally configure the timeout value (in minutes) for the micro benchmark test stage of the pipeline | 50 |
| `INTEGRATION_TEST_TIMEOUT` | optionally configure the timeout value (in minutes) for the integration test stage of the pipeline | 120 |
| `INTEGRATION_TEST_NODE` | optionally configure the jenkins executor node for the integration test stage to be executed on | the default node used for the rest of the build |
| `CONTRACT_TEST_TIMEOUT` | optionally configure the timeout value (in minutes) for the contract test stage of the pipeline | 60 |
| `CONTRACT_TEST_NODE` | optionally configure the jenkins executor node for the contract test stage to be executed on | the default node used for the rest of the build |
| `PROVIDER_VERIFICATION_TIMEOUT` | optionally configure the timeout value (in minutes) for the provider verification stage of the pipeline | 60 |
| `PROVIDER_VERIFICATION_JOBS_LIST` | optionally configure the comma separated list of provide verification jobs for the provider verification stage of the pipeline. Each value can be one of the following 3 allowed values.<br/>1. Complete path of the job including folder and branch names.Pipeline will search for an exact match. Ex: `Lean Fnx/lean-fnx-service-contract-test/main` <br/>2. Users may omit folder name and can give job and branch as folder is automatically identified by the pipeline. Ex: `lean-fnx-service-contract-test/main` <br/>3. Users can just give the repo name. Folder is automatically identified by the pipeline and provider verification build branch would be same as consumer branch. Ex: `lean-fnx-service` | 60 |
| `BUILD_TIMEOUT` | optionally configure the timeout value (in minutes) for the 'build' stage of the pipeline | 60 |
| `UNIT_TEST_AND_STATIC_ANALYSIS_TIMEOUT` | optionally configure the timeout value (in minutes) for the 'unit test * static analysis' stage of the pipeline | 20 |
| `SECURITY_TEST_TIMEOUT` | optionally configure the timeout value (in minutes) for the 'veracode' & 'whitesource' stages of the pipeline | 120 |
| `RELEASE_ANNOUNCEMENT_WEBEX_SPACE_NOTIFICATION_LIST` | comma separated list of webex space ids to receive release notifications | |
| `RELEASE_ANNOUNCEMENT_EMAIL_NOTIFICATION_LIST` | comma separated list of emails to receive release notifications | |
| `BUILD_STATUS_WEBEX_SPACE_NOTIFICATION_LIST` | comma separated list of webex space ids to receive build notifications | |
| `BUILD_STATUS_EMAIL_NOTIFICATION_LIST` | comma separated list of emails to receive build notifications | |
| `WHITESOURCE_SCAN_ENABLED` | Enable Whitesource security scanning.  Requires that your root project applies the [com.pega.securitybase](https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/security-base-plugin) -or- [com.pega.veracode](https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/veracode-plugin) plugin | false |
| `SKIP_END_TO_END_DEPLOY_IN_CI_BUILD` | skip end to end deploy in ci build (MUST be set to true if using a separate dedicated End-to-End deployment pipeline) | false |
| `SKIP_VERACODE_SCAN_IN_CI_BUILD` | skip veracode scan in CI build (MUST be set to true if using a separate security scanning pipeline) | false |
| `SKIP_WHITESOURCE_SCAN_IN_CI_BUILD` | skip whitesource scan in CI build (MUST be set to true if using a separate security scanning pipeline) | false |
| `SKIP_COMPONENT_TEST_IN_PR_BUILD` | skip component test in pr build | false |
| `SKIP_CONTRACT_TEST_IN_PR_BUILD` | skip contract test in pr build | false |
| `SKIP_PROVIDER_VERIFICATION_STAGE_IN_PR_BUILD` | skip provider verification stage in pr build | true |
| `SKIP_INTEGRATION_TEST_IN_PR_BUILD` | skip integration tests in pr build | false |
| `SKIP_INTEGRATION_TEST_IN_CI_BUILD` | skip integration tests in ci build | false |
| `SKIP_MICRO_BENCHMARK_TEST_IN_PR_BUILD` | skip micro benchmark tests in pr build | false |
| `USE_AGILE_STUDIO_RELEASE_GATE` | tag promotion builds will be marked unstable if the agile studio release record provided in the project's readme file does not have a Resolved-Active status | false |
| `UNIT_TEST_COVERAGE_REQUIRED` | optionally override unit test coverage End2End preintegration gate threshold | 60 |
| `CONTRACT_TEST_NUMBER_REQUIRED` | optionally override contract test number End2End preintegration gate threshold | 1 |
| `CONTAINER_IMAGE` | Custom container image used to execute jenkins job | docker-release-local.bin.pega.io/java-11-with-build-tools:latest |

For more details on pipeline configuration and continuous delivery model please refer [here](https://git.pega.io/projects/SQUID/repos/jenkins-service-pipeline-template/browse/templates/template/docs/cdModel.md).

