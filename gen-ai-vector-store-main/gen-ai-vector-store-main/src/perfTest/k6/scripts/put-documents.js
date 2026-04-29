import http from 'k6/http';
import {check, sleep} from 'k6';
import {randomWord} from './helpers.js';


// Read target URL from environment variable or use default
const TARGET_URL = __ENV.TARGET_URL || 'http://localhost:28080';
const ENDPOINT = '/v1/iso-integ/collections/injection-test/documents';
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
    const payload = JSON.stringify(generateDocumentPayload());

    const headers = {
        'Content-Type': 'application/json',
        ...(SAX_TOKEN ? {'Authorization': `Bearer ${SAX_TOKEN}`} : {})
    };
    const response = http.put(`${TARGET_URL}${ENDPOINT}`, payload, {headers});

    const ok = check(response, {
        'is status 200, 201, or 202': (r) => r.status === 200 || r.status === 201 || r.status === 202,
    });
    if (!ok) {
        console.log(`Request failed with status code: ${response.status}\n${response.body}`);
    }
    sleep(1); // Sleep for 1 second between requests
}

// Generate different document types for variety in testing
function generateDocumentPayload() {
    const documentId = `DOC-${randomWord(8)}-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
    const documentTypes = ['technical', 'business', 'tutorial', 'reference'];
    const docType = documentTypes[Math.floor(Math.random() * documentTypes.length)];

    // Use FIXED_NUM_CHUNKS env variable if set and valid, else random (1-4)
    // const fixedNumChunks = Math.floor(Math.random() * 4) + 1
    // const numChunks = (fixedNumChunks && Number.isInteger(fixedNumChunks) && fixedNumChunks > 0) ? fixedNumChunks : Math.floor(Math.random() * 4) + 1;
    const numChunks = 100
    const chunks = [];

    for (let i = 0; i < numChunks; i++) {
        chunks.push({
            "content": `${docType} document chunk ${i + 1} about ${randomWord()}. This section contains detailed information about ${randomWord()} and ${randomWord()}. It covers ${randomWord()} implementation details and ${randomWord()} best practices for modern applications.`,
            "attributes": [
                {
                    "name": "chunkType",
                    "type": "string",
                    "value": [i === 0 ? "introduction" : i === numChunks - 1 ? "conclusion" : "content"]
                },
                {
                    "name": "section",
                    "type": "string",
                    "value": [`section_${i + 1}_${randomWord(6)}`]
                },
                {
                    "name": "chunkIndex",
                    "type": "string",
                    "value": [i.toString()]
                }
            ],
            "metadata": {
                "embeddingAttributes": ["chunkType", "section"]
            }
        });
    }

    return {
        "id": documentId,
        "chunks": chunks,
        "attributes": [
            {
                "name": "dataSource",
                "type": "string",
                "value": [
                    "performance_test_data",
                    `${randomWord()}_documentation`,
                    `test_${randomWord()}`,
                    `${docType}_docs`
                ]
            },
            {
                "name": "roles",
                "type": "string",
                "value": [
                    "KnowledgeBuddy:Admin",
                    "KnowledgeBuddy:Public",
                    "KnowledgeBuddy:TestUser",
                    `TestRole:${randomWord()}`,
                    `${docType}:Reader`
                ]
            },
            {
                "name": "version",
                "type": "string",
                "value": [`${Math.floor(Math.random() * 10) + 1}.${Math.floor(Math.random() * 10)}.0`, "test"]
            },
            {
                "name": "category",
                "type": "string",
                "value": [`category_${randomWord(6)}`, "performance_test", docType]
            },
            {
                "name": "priority",
                "type": "string",
                "value": [["high", "medium", "low"][Math.floor(Math.random() * 3)]]
            },
            {
                "name": "documentType",
                "type": "string",
                "value": [docType]
            },
            {
                "name": "random_embeddings",
                "type": "string",
                "value": [Math.floor(Math.random() * 1000000).toString()]
            }
        ],
        "metadata": {
            "embeddingAttributes": ["dataSource", "category", "version", "documentType"],
            "extraAttributesKinds": ["auto-resolved", "index", "static"]
        }
    };
}
