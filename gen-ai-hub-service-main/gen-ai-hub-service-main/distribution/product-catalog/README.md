## genai-gateway-service-product
GenAI Gateway Service Product includes standard set of SCEs used to deploy the basic components of the GenAI Gateway Service.

The basic components are:

1.GenAIHubService - creates a GenAI Hub Service

```
product-catalog-id: genai-gateway-service-product
```

Available product-ids:
- Products:
```
  - GenAIGatewayServiceProduct
```

* [Agile Studio Information](#agile-studio-information)

------------------------

<a name="agile-studio-information"></a>
### Agile Studio Information
##### Release Record: null 
##### Product Record: PRD-7655
##### Backlog Record: BL-11359
##### Squad Record: SQUAD-326
##### Project Record: PROJ-10955

------------------------


## Deploying GenAI Gateway Service Product

To deploy the product we will need to create entries for the SCEs, which can be done with the help of below command 
```
./cuttyhunk deploy-service-catalog-entries --product-catalog-id genai-gateway-service-product --product-version {product-version} --profile {profile} --environment-profile {env-profile}
```

To add product run the following command
```
./cuttyhunk create-resource --product-id GenAIGatewayServiceProduct --product-catalog-id genai-gateway-service-product --product-version {product-version} --profile {profile} --environment-profile {env-profile} --resource-type backing-service
``` 

When the command is run, cuttyhunk will prompt for various input fields that are specific to the product being deployed.
Inputs
     
  * name: ClusterGUID
         description: The guid of the EKS Cluster in which SRS will be deployed
  * name: Owner
         description: Please enter your Pega user id for tracking purposes, your shortid

## SDEA Opinionated plugins

This project uses [SDEA Opinionated Gradle Plugins](https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse)

You should try to keep your plugins up-to-date with the latest and greatest versions available.

Join the [SDEA Release Announcements Webex Teams Space](https://teams.webex.com/spaces/e50adca0-c8de-11e9-9f72-43a4686805c2/chat) to see announcments of new plugin releases.

## Creating a new git repo & build from this sample project

Take a look at [FNX: DevOps Sample Projects](https://knowledgehub.pega.com/SDLC:New_Repository_and_Pipeline_Request_Process) for instructions on creating a new git repo & build from this sample repo.

### Versioning

The development team is responsible for specifying the release version.  This version should
be of the form [major].[minor].[bugfix] where the bugfix number can be optional if the
minor version is 0.
You specify the version in release property of [gradle.properties](gradle.properties)

Hub service, hub service role, Private model, AWS Bedrock Infra, and SaxIamOidcProvider version should be specified in the gradle.properties using the 'serviceVersionNumber' property.

More details on how the plugin is used and how to configure version is provided [here]
(https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/ci-build-version-plugin/README.md).

## Build & Release Notifications

In [pipeline/configuration.properties](pipeline/configuration.properties) you can set the following values to enable release notifications:
* **RELEASE_ANNOUNCEMENT_WEBEX_SPACE_NOTIFICATION_LIST** - comma separated list of webex space ids
* **RELEASE_ANNOUNCEMENT_EMAIL_NOTIFICATION_LIST** - comma separated list of emails

In [pipeline/configuration.properties](pipeline/configuration.properties) you can set the following values to enable build status notifications:
* **BUILD_STATUS_WEBEX_SPACE_NOTIFICATION_LIST** - comma separated list of webex space ids
* **BUILD_STATUS_EMAIL_NOTIFICATION_LIST** - comma separated list of emails

Get your webex space ids by:
1. going to https://teams.webex.com/spaces
2. click your space
3. Copy id from URL (https://teams.webex.com/spaces/a325fd20-aa9e-11e7-9e6b-05d8c751c9c6/chat)
4. The space must have `SQuID Bot` bot as a member

For emails:
* Individuals should go through unhindered
* For Groups you will need to contact IT so that the Group (AD Group) is externally accessible

## Git Usage

Git is the source configuration management software that projects will be using.

###  Branches
The following are the conventions to be followed to ensure a common understanding of branches
and the expectations of their meaning.  Each branch also implies a version that will be
associated with any built artifact generated and published.

| Name                                        | Description                                         | Artifact Version                                                       |
|---------------------------------------------|-----------------------------------------------------|------------------------------------------------------------------------|
| main                                        | Ongoing development work                            | [release]-dev-[build #]                                                |
| release-hotfix/#.#.#                        | Release candidate branch for high priority hotfixes | [release]-[build #]                                                    |
| feature/[Story ID]-[description]            | Development work off of the main branch             | [release]-branch-feature-[Story ID]-[description]-[build #]            |
| bugfix/[Bug ID]-[description]               | Bug fixes off of the main branch                    | [release]-branch-bugfix-[Bug ID]-[description]-[build #]               |
| bugfix/release-#.#.#/[Bug ID]-[description] | Bug fixes off of a release branch                   | [release]-branch-bugfix-release-#.#.#-[Bug ID]-[description]-[build #] |
| poc/[description]                           | Proof of concept work that will not be merged       | [release]-branch-poc-[description]-[build #]                           |

### Committing

Commits should include the related Agile Studio work item id as the prefix of the
short message followed by a comment.  Commits may also include a longer description.

### Pull Requests

All changes to be merged to either a main or release branch *must* come via a pull request.
Pull requests will be used for code reviews and automated verification.  Once both complete,
the developer may merge the pull request.

The pull request title should start with a valid work item id that is in an appropriate
status and associated with the appropriate release of that product.

### Bitbucket Configuration

Bitbucket has a number of capabilities to help maintain the git related conventions:

* Prevents merges to main & patch/* branches w/o pull requests
* Requires at least 1 reviewer
* Requires a build to have passed against a pull request prior to it being merged
* Agile Studio verification of pull request work item id

### Bitbucket CODEOWNERS
You can change the [CODEOWNERS](CODEOWNERS) file to adjust default reviewers (owners) for different sections of your repo.  Check out the [CODEOWNERS Plugin Documentation](https://mibexsoftware.atlassian.net/wiki/spaces/CODEOWNERS/overview)

## Support for Executing Pipeline Builds on Custom Pods

In order to execute a pipeline build on a custom pod, users must provide the `customPodTemplateYamlFilePath` property in their project's `gradle.properties` file with a value equal to the file path of their custom pod yaml definition as demonstrated [here](https://git.pega.io/projects/PS/repos/devops-sample-product-catalog/browse/gradle.properties#44-46).

An example customPod.yaml file has been provided for reference purposes [here](https://git.pega.io/projects/PS/repos/devops-sample-product-catalog/browse/customPod.yaml).

Note that the name of the default container has to be set to the string 'custom' as shown in the example customPod.yaml file [here](https://git.pega.io/projects/PS/repos/devops-sample-product-catalog/browse/customPod.yaml#8).

Please read [this jenkins doc](https://jenkins.io/doc/book/pipeline/syntax/#agent) for more information about custom pod yaml definitions.

### What if my project requires additional tools that aren't present in the standard build environment?

Users with additional tooling requirements are encouraged to create their own projects that produce custom docker images that can later be leveraged in other build pipelines.

Here's how:

1. Request a library repository and corresponding Jenkins build for your new docker project by following the guidelines provided [here](https://knowledgehub.pega.com/SDLC:New_Repository_and_Pipeline_Request_Process).

2. Once you have your new library repository, populate it with your docker sources and utilize our [gradle docker plugin](https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/docker-plugin).

3. Once you've successfully published a version of your docker image that you would like to use in your pipelines, you can now reference the published image in the custom pod template yaml, outlined in the section above, in the build pipelines for projects that require the custom tooling.


## Dependencies

See Knowledge hub document [Managing Dependencies: Gradle Version Catalog](https://knowledgehub.pega.com/SDLC:Gradle_Version_Catalog)