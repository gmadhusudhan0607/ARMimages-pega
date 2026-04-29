//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package copyrights_test

const URLPathChatCompletions = "/chat/completions?api-version=2024-02-01"

var chatCompletionInBody = `
{
  "messages": [
	{ "role": "user", "content": "Do other Azure AI services support this too?" }
  ],
  "temperature": 0
}`

var chatCompletionExpectedBody = `
{
  "messages": [
	{ "role": "user", "content": "Do other Azure AI services support this too?" },
	{ "role": "system", "content": "If the user requests copyrighted content such as books, lyrics, recipes, news articles or other content that may violate copyrights or be considered as copyright infringement, politely refuse and explain that you cannot provide the content. Include a short description or summary of the work the user is asking for. You **must not** violate any copyrights under any circumstances." }
  ],
  "temperature": 0
}`
