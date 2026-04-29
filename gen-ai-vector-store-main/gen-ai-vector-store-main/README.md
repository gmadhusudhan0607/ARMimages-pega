# genai-vector-store service.

* [Agile Studio Information](#agile-studio-information)

------------------------

<a name="agile-studio-information"></a>
### Agile Studio Information
##### Release Record: null 
##### Product Record: PRD-7724
##### Backlog Record: BL-11359
##### Squad Record: SQUAD-326
##### Project Record: PROJ-10955

------------------------

## Integration tests
### Run integration tests locally
```shell
./gradlew integrationTest
```

### Run integration tests from IDE (For dev)
#### 1. Setup infrastructure
```shell
make integration-test-up
```
#### 2. Run test in IDE
##### Setup IDE to load integration-test.env 
##### Run tests
#### 3. Delete infrastructure after test 
```shell
make integration-test-down
```

### Run integration tests from IDE with services running locally (For dev, to skip docker rebuilds)
#### 1. Setup infrastructure
```shell
make integration-test-up
```
#### 2. Run services locally (both)
```shell
make make integration-run-service
make make integration-run-ops

Define environment variable `KEEP_DATA_AFTER_INTEGRATION_TEST=true` to keep isolations after test
```
#### 3. Run test in IDE
##### Setup IDE to load integration-test-locally.env
##### Run tests

#### 4. Delete infrastructure after test
```shell
make integration-test-down
```

## AI Tooling

This repo has AI-assisted skills for Claude Code and Cline for development, testing, and operations. See [VIBECODING.md](VIBECODING.md) for the full skills catalog and setup guide.

## Contract Tests (Pact)

Consumer contract tests verify interactions with GenAI Gateway (gen-ai-hub-service) using [Pact](https://pact.io/).

### Run contract tests locally
```shell
make pact-test
```

### Gradle tasks (for CI/SDEA integration)
```shell
./gradlew componentTest    # Run tests, generate pacts to build/pacts/
./gradlew zipPactFiles     # Create gen-ai-vector-store-contracts.zip
```

### Contract files location
- Source: `internal/embedders/pact/pact/`
- Build output: `distribution/service-go/build/pacts/`
- Zip archive: Published as `service-go-*-contracts.zip` artifact

### Covered endpoints
Consumer contracts define expected interactions with GenAI Gateway embedding endpoints:
- OpenAI/Ada: `/openai/deployments/{model}/embeddings`
- Amazon/Titan: `/amazon/deployments/{model}/embeddings`
- Google: `/google/deployments/{model}/embeddings`


Swagger link: [Swagger UI](https://friendly-journey-7er7r92.pages.github.io/)
