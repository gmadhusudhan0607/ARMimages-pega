## Running the integration test locally

<br/>

### What do you need to run the integ tests locally?

<br/>**GitHub Token (private Go module)**
<p>The service depends on <code>github.com/Pega-CloudEngineering/go-sax</code>, a private Go module. Fetching it during the Docker build requires a GitHub personal access token with read access to the <code>Pega-CloudEngineering</code> organisation.

If you are authenticated with the GitHub CLI, retrieve your token with:
<pre>gh auth token</pre>

Prefer exporting the credentials only for your current shell session instead of persisting them in your shell profile (for example <code>~/.bashrc</code> or <code>~/.zshrc</code>). Only add them to a profile if you explicitly need that behaviour:
<p><code>GITHUB_PSW</code> is the existing variable name expected by the build, but its value must be a GitHub personal access token (PAT), not your GitHub account password.</p>
<pre>
export GITHUB_USR=&lt;your-github-username&gt;
export GITHUB_PSW=&lt;your-github-personal-access-token&gt;
</pre>

To allow <code>go mod download</code> to resolve the module outside of Docker, add an entry to <code>~/.netrc</code>:
<pre>machine github.com login &lt;your-github-username&gt; password &lt;your-github-token&gt;</pre>
<p>Because <code>~/.netrc</code> contains a plain-text token, do not commit or share it, and restrict its permissions so only your user can read and write it:
<pre>chmod 600 ~/.netrc</pre>

Alternatively, you can rewrite GitHub HTTPS URLs to embed credentials so Go module fetches authenticate automatically:
<pre># Rewrite github.com HTTPS URLs to include credentials so Go module fetches authenticate.
git config --global url."https://${GITHUB_USR}:${GITHUB_PSW}@github.com/".insteadOf "https://github.com/"</pre>

<br/>**Java**
<p>Gradle 8.3 requires Java 17. If your default <code>java</code> is a newer version, set <code>JAVA_HOME</code> to point at a Java 17 installation, update <code>PATH</code> so <code>java</code> resolves from that JDK, and add both to your shell profile:
<pre>export JAVA_HOME=&lt;path-to-java-17&gt;
export PATH="$JAVA_HOME/bin:$PATH"</pre>

<br/>**Docker**
<p>Check if docker is running. You can check running <pre>docker ps</pre> or <pre>docker info</pre>

If you are on WSL on Windows, you may need to enable your distribution in the Docker Desktop. Check if your distribution is enabled under *Docker Desktop > Settings > Resources > WSL Integration*. The checkbox for your Default WSL distro must be checked, and if you use an additional distro you need to toggle it on.

<br/>**Artifactory (binbos.pega.io)**
<p>The docker images are stored in https://binbos.pega.com. This registry requires authentication. To authenticate you can login to Artifatory using SAML. After logged in, click on *Set me Up* under your profile Id in the right top of your screen. Click on *Docker* and copy the snippet code that need to be pasted to the ~/.docker/config.json

Change the host name on the 2nd line to "binbos.pega.com:5002" and test it on your terminal:

<pre>
$ docker login binbos.pega.com:5002
Authenticating with existing credentials...
WARNING! Your password will be stored unencrypted in /home/capcloud/.docker/config.json.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store

Login Succeeded
</pre>

<br/>**Docker-Dev (docker-dev.bin.pega.io)**

It is the same procedure that followed for **binbos.pega.com** above, but you need to create your toke on https://bin.pega.io instead. Log in and on the *Set me up* choose Docker, and the Repository the Docker-Dev in the dropdown on the next screen. 

It will display an HTTP Token, and below the json snipped to be added to the ~/.docker/config.json file. Add the authentication to the file and test with docker login:

<pre>
$ docker login docker-dev.bin.pega.io
Authenticating with existing credentials...
WARNING! Your password will be stored unencrypted in /home/capcloud/.docker/config.json.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store

Login Succeeded
</pre>

Alternatively you can instead of add the servers to ~/.docker/config.json you can add then to the ~/.gradle/gradle.properties, like the example below:
<pre>
dockerRegistry=docker-dev.bin.pega.io
dockerRegistryUser=${mypegaid}
dockerRegistryPassword=${httptoken}
</pre>

<br/>**Helm**
<p>To set up Helm integration it is the same procedure. Add to ~/.gradle/gradle.config:

<pre>
helmRepoUsername=pegaid
helmRepoPassword=token
</pre>

to generate the Token, use the option *Set me up* under your profile name in Artifactory.

## Executing the tests

The following task will setup, run and tear down the integration test resources.
<pre>
$ ./gradlew integrationTest
</pre>

Each step also can be executed individually with these tasks:
<pre>
integrationTestUp
integrationTestRun (will trigger integrationTestDown automatically)
integrationTestDown
</pre>

In the same way the tests can be done using the `make` command.

<pre>
integration-test-up
integration-test-run
integration-test-down
</pre>

The make task `integration-test-run` do not trigger the `integration-test-down`, keeping the containers up. 

## Integration Tests architecture

This integration test suite runs locally in your machine the `genai-hub-service` container so its 
API can be exercised locally. 

The genai-hub-service requires a destination service (GenAI Model Deployment) to be available. As this is a local test 
and we dont have a GenAI Model running locally, it is configured a `mockserver` container which is beint used
as a double for any external service integration that is required for the `genai-hub-serice` to execute.

To orchestrate the start of the containers it is being used `docker-compose` tool instead of any Kubernetes 
distribution to facilitate the setup, since putting a kubernetes cluster locally would demand much more 
setup. *If anybody is up for a challenge, here is a good one - run the integration tests on a local Kubernetes 
distro like minikube or k3s.*

Although, as the Service is not being provisioned using an SCE, the container parameterization need to be done manually.
The required configurations for a `genai-hub-service` to execute locallly are (minimal):
- a mapping file (mapping.yaml) to be on a volume mounted to `/config` path
- the environment variables that are require during runtime in the docker-compose configuration

The mapping files are dynamically generated based on the mapping template used in the project. The mapping is 
generated using a make task, called `generateMappingFiles`. The Makefile is located in the root
of the project.

When using docker, the philosophy is that containers should be *immutables*. That is a best
practice also for testing, as changing a running environment may cause unpredicted side effects due
to the order tests are executed or concurrent test execution. To avoid these, everytime a new test scenario
that a specific configuration is needed, instead of modify one of the containers we aim to 
add a new container to the `docker-compose` to be executed. If necessary, a new instance of `mockserver` also
can be added, so tests do not influence each other.

Every new container added should be exposed to a different local port. The port that will be used will
be added to the mapping file.

### Other dependencies - AWS, UAS and OKTA

For the launchpad scenario the `genai-hub-service` need to have other dependencies added.

#### AWS

For AWS the dependency is due to the need to fetch secrets from AWS Secret Manager. The AWS endpoint 
can be mocked using environment variables in the running container to point the AWS SDK/CLI
to issue their requests against a custom endpoint.

```
AWS_IGNORE_CONFIGURED_ENDPOINT_URLS=false # AWS SDK/CLI to allow custom endpoints
AWS_ENDPOINT_URL=http://localhost:1108    # mockserver URL
```
https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html


The mocking of individual AWS endpoints can be done in a more granular fashion as well, using 
a notation that adds the service at the end of the environment variable key, like:
```
AWS_ENDPOINT_URL_STS
AWS_ENDPOINT_URL_SECRETS_MANAGER
AWS_ENDPOINT_URL_BEDROCK_RUNTIME
```
https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html#envvars-list-AWS_ENDPOINT_URL_SERVICE


#### UAS

UAS is an authentication and authorization system using by Launchpad microservices. It is a service
deployed to the same cluster then the genai-hub-service.

It could be mocked using the `mockserver` as it is expected an API call to be done and answered. Although, UAS 
code depends on encryption and other checks, making it difficult to put a mocked response that ensure all code works.
*Maybe* having the entire UAS code mocked using `mockgen` would be a simpler approach

*NOTE: Check which mock response could work with the UAS code implemented. Preferably we want to mock with the mockserver.* 

#### OKTA

OKTA is an external endpoint tha will provide a new token and the ability to sign a request. Even though the 
request and response structure may be complex, we have more access to create this data.
As it is an external endpoint, we can mock it on the mockserver, or instead mock the 
external package we have imported that do the integration on our behalf with Okta Auth server.

## How to create a test scenario

The `mockserver` works in the following manager:
- an expected call need to be defined, providing the Path and, optionally, how the request looks like
- the expectation need to have what the response should be, so it will be served every time that path is reached
- the verification of the expectation, if what was expected really happened

With that, when writing a new test scenario it is necessary to always set the mockserver expectation
in advance, and later remove that expectation from the server to avoid affect other test scenarios.

To execute the tests is used a took called `ginkgo`. The `ginkgo` command is run when the task
`integration-test-run` is triggered from the Makefile, or by using the gradle task `integrationTestRun`. For
the integration tests, the gradle tasks are basically proxying the execution to Makefile 
instead of using its default plugin.