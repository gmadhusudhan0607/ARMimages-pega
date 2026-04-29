import http from 'k6/http';
import {check, sleep} from 'k6';
import {randomWord} from './helpers.js';


// Read target URL from environment variable or use default
const TARGET_URL = __ENV.TARGET_URL || 'http://localhost:80';
const ENDPOINT = '/v1/iso-integ/collections/col-1m/query/chunks';
const SAX_TOKEN = open('/etc/secrets/SAX_TOKEN', 'r').trim();

if (!SAX_TOKEN) {
    throw new Error('Auth token file /etc/secrets/SAX_TOKEN is required but not provided or empty.');
}

const HTTP_DURATION_THRESHOLD_MS = Number(__ENV.HTTP_DURATION_THRESHOLD_MS) || 5000;
const HTTP_DURATION_PERCENTILE = Number(__ENV.HTTP_DURATION_PERCENTILE) || 95;
const VIRTUAL_USERS = Number(__ENV.VIRTUAL_USERS) || 1;
const TEST_DURATION = __ENV.TEST_DURATION || '3s';

export const options = {
    vus: VIRTUAL_USERS,
    duration: TEST_DURATION,
    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: [`p(${HTTP_DURATION_PERCENTILE})<${HTTP_DURATION_THRESHOLD_MS}`],
    },
};


export default function () {
    const payload = JSON.stringify({
        "limit": 5,
        "maxDistance": 2,
        "retrieveAttributes": [
            "random_embeddings"
        ],
        "filters": {
            "query": `what is ${randomWord()}`,
            "attributes": [
                {
                    "name": "random_embeddings",
                    "type": "string",
                    "value": [
                        "200000"
                    ]
                },
                {
                    "name": `dataSource`,
                    "type": "string",
                    "value": [
                        `databricks_documentation`
                    ]
                },
                {
                    "name": `roles`,
                    "type": "string",
                    "value": [
                        `KnowledgeBuddy:Admin`, `KnowledgeBuddy:Public`, `KnowledgeBuddy:BuddyManager`,
                        `KnowledgeBuddy:DataSourceManager`, `KnowledgeBuddy:Internal`
                    ]
                }
            ]
        }
    });
    const headers = {
        'Content-Type': 'application/json',
        ...(SAX_TOKEN ? {'Authorization': `Bearer ${SAX_TOKEN}`} : {})
    };
    const response = http.post(`${TARGET_URL}${ENDPOINT}`, payload, {headers});

    const ok = check(response, {
        'is status 200': (r) => r.status === 200,
    });
    if (!ok) {
        console.log(`Request failed with status code: ${response.status}`);
    }
    sleep(1); // Sleep for 1 second between requests

}