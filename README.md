# AiResponder
Playground for Ai and Web Application Security

## Crawl
**crawl** uses playwright-go to crawl endpoints on a given URL and *log* all headers and content for Ai to analyze for web application security vulnerabilities and potential bugs. This is to attempt to automate some vulnerability analysis and give bug hunters an efficient way to find vulnerabilities.

### Usage
```bash
Usage of crawl:
  -depth int
    	maximum crawl depth (default 3)
  -timeout int
    	request timeout in seconds (default 30)
  -url string
    	target URL to crawl (e.g., 'http://example.com')
```

### Prerequisite
1. Install playwright-go
```bash
go mod init main
go mod tidy
```

2. Install browser(s) for playwright-go
```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
# Or
go install github.com/playwright-community/playwright-go/cmd/playwright@latest
playwright install --with-deps
```

### Compile
```bash
go build -o crawl crawl.go
```

### logs
All headers are logged to a json file *logs.json* in a format to easily parse
```json
[
  {
    "url": "https://example.com/wp-content/plugins/wp-rocket/assets/js/wpr-beacon.min.js",
    "requests": [
      {
        "method": "GET",
        "url": "https://example.com/wp-content/plugins/wp-rocket/assets/js/wpr-beacon.min.js",
        "headers": {
          "accept": "*/*",
          "accept-encoding": "gzip, deflate, br, zstd",
          "accept-language": "en-US,en;q=0.5",
          "connection": "keep-alive",
          "host": "example.com",
          "referer": "https://example.com/shop/",
          "sec-fetch-dest": "script",
          "sec-fetch-mode": "no-cors",
          "sec-fetch-site": "same-origin",
          "user-agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:132.0) Gecko/20100101 Firefox/132.0"
        }
      }
    ],
    "responses": [
      {
        "url": "https://example.com/wp-content/plugins/wp-rocket/assets/js/wpr-beacon.min.js",
        "status_code": 200,
        "headers": {
          "accept-ranges": "bytes",
          "age": "84924",
          "cache-control": "max-age=31622400",
          "content-encoding": "gzip",
          "content-length": "4081",
          "content-type": "application/x-javascript",
          "server": "nginx",
          "strict-transport-security": "max-age=300",
          "vary": "Accept-Encoding",
        },
        "body": ""
      }
    ],
    "content": ""
  },
]
```

## Gemini Analyzer
**Gemini-analyzer** is a BurpSuite Extension used to send requests/responses to gemini-1.5 within burpsuite directly. This is to assist in finding potentially abstract or missed vulnerabilities within given headers or response bodies.

### Prerequisite
1. Gemini API key in *gemini-analyzer.py*
```python
GEMINI_API = "GEMINI_API_KEY"
```

### Usage
It's important to take note of prompting the Gemini Ai in more detail for efficiency in what you're analyzing. It's best to be more specific with vulnerabilities one is looking for and use such as a *microservice* for finding them effectively.
- see the *gemini-analyzer.py* file to **edit** the prompt.
