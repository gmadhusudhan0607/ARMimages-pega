Performance Tests
=================

# Setup
Peformance tests are executed on the cluster and genai-vector-store instance pointed by gradle.properties

## Gradle properties:
| Variable                          | Description                                   | 
|-----------------------------------|-----------------------------------------------|
| performanceEnvironmentProfile     | Environment profile, e.g. integration         |
| performanceResourceType           | Resource type. Here: backing-service          |
| performanceResourceGuid           | GenAIVectorStore instance GUID on the cluster              | 
| performanceClusterGuid            | Cluster GUID                                  |
| performanceServiceEndpointURL     | GenAIVectorStore URL (accessible from the jmeter) |
| performanceIsolationId            | GenAIVectorStore API IsolationId |
| performanceCollectionId            | GenAIVectorStore API CollectionId  |

## Cluster setup
Make sure that:
- Geani-vector-store is running and accessible from outside of the cluster. TODO: confirm below:
To set it accessible update the isFrontendService property of the service in GOC

## Local setup

### Install prerequisities

#### bzt
On Mac
```
brew install bzt 
```
On Linux
```shell
sudo python3 -m pip install bzt
```

To run the tests locally, you might need to create account at blazemeter.com, generate api keys and place them in the ~/.bzt-rc file in a form:
```
modules:
  blazemeter:
    token: <key id>:<key secret>  # API id and API secret joined with ':'
```


### Connect to okta

Get valid sax token.
Currentlly the workaround is to connect to genai profil (see below)
```
ok genai
```
or
```
okta-awscli --okta-profile genai --profile genai --force
```

Where genai profile points to:

```properties
[genai]
username = maciej.surdziel@pega.com
app-link = https://pega.okta.com/home/amazon_aws/0oam6u4wyuVYgbLdJ0x7/272
base-url = pega.okta.com
duration = 43200
profile = hoc
role = arn:aws:iam::100167087043:role/okta_poweruser

[genai2]
username = maciej.surdziel@pega.com
app-link = https://pega.okta.com/home/amazon_aws/0oar1iprr821hO5Yi0x7/272
base-url = pega.okta.com
duration = 43200
profile = hoc
role = arn:aws:iam::045663071481:role/okta_poweruser
```

Then in the folder with cloned https://it.pega.io/scm/pcld/go-sax.git issue a command:
```
go run . issue --secret-id arn:aws:secretsmanager:us-east-2:100167087043:secret:sax/backing-services/458d4701-79c5-4343-bcca-9eaa69d1cedf-apXJUZ --region us-east-2
```
Set the output Access Token in gradle.properties performanceTokenhack property

### Run tests
#### Baseline
To run perfTest from laptop with the default configuration (as per gradle.properties) run:
```
./gradlew perfTest --gui
```


It opens Jmeter. To see the exact requests in Jmeter, right click on "View results tree" and "enable".
Run test (green triangle) and observe req/resp assertions in "View Results Tree"

#### Scale
To run scenarios defined in resiliencyTestsEnabled (by default - Scale) from laptop with the default configuration (as per gradle.properties) run:
```
./gradlew resilTest --gui
```

### Modify tests
In order to modify the tests, first Run them.
On startup it creates copy of src/perfTest/genai-vector-store.jmx into modified_genai-vector-store.jmx.
So if you make any changes to the Jmeter configuration, make sure to copy content of the  modified_genai-vector-store.jmx into genai-vector-store.jmx before closing the app, otherwise the changes will be lost.

## Analyze test results

TODO: The latest builds can be found at https://ci.pega.io/cloud-services/job/gen-ai-vector-store/job/gen-ai-vector-store-performance-test/job/main/

### Metrics
Response times for the following endpoints/processes are measured:
| URI | Description | 
|-----|-------------|
| PUT documents   | Request to PUT /v1/{{isolation_id}}/collections/{{collection_id}}/documents |
| POST query/chunks | Request to POST /v1/{{isolation_id}}/collections/{{collection_id}}/query/chunks |
| GET documents | Request to GET /v1/{{isolation_id}}/collections/{{collection_id}}/documents |
| DELETE documents | Request to DELETE /v1/{{isolation_id}}/collections/{{collection_id}}/documents |
