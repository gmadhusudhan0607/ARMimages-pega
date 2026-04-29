#Getting started with GenAI Vector Store

##1. Setting up Infinity:

Create a DSS:
```
prconfig/services/genai/vectorstore/servicebaseurl/default

```
##2. Setting up your laptop
**1. Pre requisites**

- Install golang >= v1.25 if not installed
- Install docker compose

- Be sure te be logged in into cirrus artifactory to be able to download docker image via docker-compose.yaml. Please take a look at https://knowledgehub.pega.com/CLDRLSEN:Authentication_and_Access_for_Cirrus_(Artifactory)_for_end_users#Generating_Token_to_access_Cirrus and then example login procedure https://knowledgehub.pega.com/CLDRLSEN:Authentication_and_Access_for_Cirrus_(Artifactory)_for_end_users#Docker

**2. Checkout Vector Store Service repository**

```commandline
ssh://git@git.pega.io:7999/pcld/gen-ai-vector-store.git
```

Note: in case of encountering permission errors, make sure you have the right permissions to the repo - please raise a request for following ```Bitbucket-Prod-Pega Cloud-WRITE``` via https://sailpoint.pega.com/

**3. connect to Azure Openai**
- Log in to Azure Portal
- If you don't have access to azure openai request it using the link https://pegasystems.sharepoint.com/sites/sp-sysops/Shared%20Documents/Forms/AllItems.aspx?id=%2Fsites%2Fsp%2Dsysops%2FShared%20Documents%2FSysOps%20%2D%20All%20Pega%2FAll%20Pega%20Documentation%2FAzure%2FAzure%20Open%20AI%20Governance%2Epdf&parent=%2Fsites%2Fsp%2Dsysops%2FShared%20Documents%2FSysOps%20%2D%20All%20Pega%2FAll%20Pega%20Documentation%2FAzure&p=true&ga=1

- Go to -> Azure OpenAI -> Navigate to one of the accounts/regions with Ada model -> Navigate to "Keys and Endpoint" section from left hand side menu (under Resource Management) ->
Copy one of the API Keys shown there - inject it into docker-compose.yaml (see section 4) as value for "GEN_AI_API_KEY"
Copy the Endpoint - it will be needed for the docker-compose.yaml (see section 4) for GENAI_GATEWAY_SERVICE_URL

Go to Networking section (left hand side menu) - add your office VPN IP (you can check it https://knowledgehub.pega.com/ITINFOPS:External-ip-addressing--nat-ips-) into Firewall section and save the new setting.

**4. Set up docker-compose.yaml**

- open the cloned GenAI Vector Store Service repo
- navigate to test/SSOT/docker-compose.yaml
- set up following env variables:

* "GENAI_GATEWAY_SERVICE_URL" - URL of gateway service
  - it should be the endpoint of the Genai-hub-service  or other gateway service that you are using. Path will be automatically appended to the URL.
```
<endpoint>/openai/deployments/<model-name>/embeddings?api-version=2023-05-15
```

example: GENAI_GATEWAY_SERVICE_URL=http://genai-hub-service.genai-hub-service.svc.cluster.local:443

* "GENAI_GATEWAY_CUSTOM_CONFIG" - overrides model URL.
example: GENAI_GATEWAY_CUSTOM_CONFIG='{"openai-text-embedding-ada-002":"https://custom-ada-endpoint.com/openai/deployments/text-embedding-ada-002/chat/completions?api-version=2023-05-15"}'

* "GENAI_API_KEY" - it should contain the value retrieved from the azure portal as stated in  Section 3.
* "SERVICE_PORT", "SERVICE_HEALTHCHECK_PORT" - if you wish the service to run on a different port as well as change the healthcheck port you can redefine it while running the the 'docker compose run -e'.

As you can observe both services start in SAX_DISABLED mode which means no authorization is required while making calls.

##3. How to bring up the service while running locally

1. run:
```
'docker compose up -d' if you wish to keep the env variables as they are or

'docker compose run -e SERVICE_PORT=8090 -e SERVICE_HEALTHCHECK_PORT=5090' to redefine some of the env variables see https://docs.docker.com/compose/environment-variables/set-environment-variables/#set-environment-variables-with-docker-compose-run---env for reference
```
to setup db and services, while being in /SSOT folder of the repo

You should be able to see three containers: vs-db-1 (status Healthy), vector-store and vector-store-ops (status Started) created and network created.

other useful commands:
```
docker compose logs --follow <container-name>
```

2. make sure you're on VPN (for Azure OpenAI model connectivity)
3. Connect to service from local ssot

In prconfig/services/genai/vectorstore/servicebaseurl/default Set value: http://localhost:8090


