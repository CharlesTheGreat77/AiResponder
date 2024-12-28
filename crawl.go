package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type LogData struct {
	URL       string        `json:"url"`
	Requests  []RequestLog  `json:"requests"`
	Responses []ResponseLog `json:"responses"`
	Content   string        `json:"content"`
}

type RequestLog struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

type ResponseLog struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body,omitempty"`
}

var logFile *os.File
var logEncoder *json.Encoder
var logEntries []LogData

func main() {
	domain := flag.String("url", "", "target URL to crawl (e.g., 'http://example.com')")
	depth := flag.Int("depth", 3, "maximum crawl depth")
	timeout := flag.Int("timeout", 30, "request timeout in seconds")
	flag.Parse()

	if *domain == "" {
		flag.Usage()
		return
	}

	Crawl(*domain, *depth, *timeout)
}

// function to init log file
func initLogFile() {
	file, err := os.OpenFile("logs.json", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("[-] Failed to open log file: %v", err)
	}
	logFile = file
	logEncoder = json.NewEncoder(logFile)
	logEncoder.SetIndent("", "  ")
}

// function to write to the log file
func writeLogs() {
	if err := logEncoder.Encode(logEntries); err != nil {
		log.Printf("[-] Error writing logs to file: %v", err)
	} else {
		fmt.Println("[*] Log data written to file.")
	}
}

// function to close the log file
func closeLogFile() {
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			log.Printf("[-] Error closing log file: %v\n", err)
		}
		fmt.Println("[*] Logs have been written to logs.json")
	}
}

// function to crawl URL(s) on a given target URL
func Crawl(domain string, depth int, timeout int) {
	visited := make(map[string]bool)

	baseURL, err := url.Parse(domain)
	if err != nil {
		log.Fatalf("[-] Invalid domain: %v", err)
	}

	baseDomain := baseURL.Host

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("[-] Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Firefox.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("[-] Failed to launch Firefox: %v", err)
	}
	defer browser.Close()

	context, err := browser.NewContext()
	if err != nil {
		log.Fatalf("[-] Failed to create browser context: %v", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		log.Fatalf("[-] Failed to create a new page: %v", err)
	}
	defer page.Close()

	initLogFile()
	defer closeLogFile()

	responseMap := make(map[string]playwright.Response)
	page.On("response", func(response playwright.Response) {
		responseURL, err := url.Parse(response.URL())
		if err != nil {
			return
		}

		if !isInScope(baseDomain, responseURL.Host) {
			return
		}

		if !visited[response.URL()] {
			visited[response.URL()] = true
			responseMap[response.URL()] = response
		}
	})

	var crawl func(string, int)
	crawl = func(currentURL string, currentDepth int) {
		if currentDepth > depth || visited[currentURL] {
			return
		}
		visited[currentURL] = true
		noFragmentURL := removeFragment(currentURL)
		fmt.Printf("[*] Crawling: %s (Depth: %d)\n", noFragmentURL, currentDepth)

		_, err = page.Goto(currentURL, playwright.PageGotoOptions{
			Timeout:   playwright.Float(float64(timeout) * 1000),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})

		if err != nil {
			fmt.Printf("[-] Failed to navigate to %s: %v\n", currentURL, err)
			return
		}

		err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateLoad,
		})
		if err != nil {
			fmt.Printf("[-] Failed to wait for load state for %s: %v\n", currentURL, err)
			return
		}

		for url, response := range responseMap {
			requestLog := RequestLog{
				Method:  response.Request().Method(),
				URL:     response.URL(),
				Headers: response.Request().Headers(),
			}
			responseBody, err := response.Body()
			if err != nil {
				log.Printf("[-] Failed to get response body for %s: %v\n", response.URL(), err)
				continue
			}

			logData := LogData{
				URL:      url,
				Requests: []RequestLog{requestLog},
				Responses: []ResponseLog{
					{
						URL:        response.URL(),
						StatusCode: response.Status(),
						Headers:    response.Headers(),
						Body:       string(responseBody),
					},
				},
				Content: string(responseBody),
			}

			logEntries = append(logEntries, logData)
		}
		responseMap = make(map[string]playwright.Response)

		// need to update [EvalOnSelectorAll is deprecated]
		links, err := page.EvalOnSelectorAll("a[href], form[action], script[src], iframe[src], img[src], link[href], meta[http-equiv=refresh][content]", `elements => {
			return elements.map(el => {
				if (el.tagName === 'META' && el.httpEquiv === 'refresh') {
					const content = el.content || '';
					const urlIdx = content.indexOf('url=');
					if (urlIdx !== -1) return content.substring(urlIdx + 4);
					return null;
				}
				return el.href || el.action || el.src || null;
			}).filter(Boolean);
		}`)
		if err != nil {
			fmt.Printf("[-] Failed to extract links: %v\n", err)
			return
		}

		for _, link := range links.([]interface{}) {
			linkStr := link.(string)
			if linkStr == "" {
				continue
			}
			absoluteURL := resolveURL(linkStr, currentURL)
			if absoluteURL == "" {
				continue
			}
			linkURL, err := url.Parse(absoluteURL)
			if err != nil {
				continue
			}
			if !isInScope(baseDomain, linkURL.Host) {
				continue
			}
			absoluteURL = removeFragment(absoluteURL)
			if !visited[absoluteURL] {
				crawl(absoluteURL, currentDepth+1)
			}
		}
	}
	crawl(domain, 0) // recursion to keep crawling
	// should I be writing logs as it crawls?
	writeLogs()
}

// function to remove fragments from URL(s)
func removeFragment(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	parsedURL.Fragment = ""
	return parsedURL.String()
}

// function to return the base URL of target [domain]
func resolveURL(relative, base string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	relativeURL, err := url.Parse(relative)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(relativeURL).String()
}

// function to stay in scope of target domain
func isInScope(baseDomain, urlHost string) bool {
	return strings.HasSuffix(urlHost, baseDomain)
}
