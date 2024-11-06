package resty

import (
	"bytes"
	"io"
	"net/http"
	"regexp"

	"net/url"
	"strings"
)

func buildCurlCmd(req *Request) string {
	// 1. Generate curl raw headers
	var curl = "curl -X " + req.Method + " "
	headers := dumpCurlHeaders(req.RawRequest)
	for _, kv := range *headers {
		curl += "-H " + cmdQuote(kv[0]+": "+kv[1]) + " "
	}

	// 2. Generate curl cookies
	// TODO validate this block of code, I think its not required since cookie captured via Headers
	if cookieJar := req.client.CookieJar(); cookieJar != nil {
		if cookies := cookieJar.Cookies(req.RawRequest.URL); len(cookies) > 0 {
			curl += "-H " + cmdQuote(dumpCurlCookies(cookies)) + " "
		}
	}

	// 3. Generate curl body
	if req.RawRequest.GetBody != nil {
		body, err := req.RawRequest.GetBody()
		if err != nil {
			req.log.Errorf("curl: %v", err)
			return ""
		}
		buf, _ := io.ReadAll(body)
		curl += "-d " + cmdQuote(string(bytes.TrimRight(buf, "\n"))) + " "
	}

	urlString := cmdQuote(req.RawRequest.URL.String())
	if urlString == "''" {
		urlString = "'http://unexecuted-request'"
	}
	curl += urlString
	return curl
}

// dumpCurlCookies dumps cookies to curl format
func dumpCurlCookies(cookies []*http.Cookie) string {
	sb := strings.Builder{}
	sb.WriteString("Cookie: ")
	for _, cookie := range cookies {
		sb.WriteString(cookie.Name + "=" + url.QueryEscape(cookie.Value) + "&")
	}
	return strings.TrimRight(sb.String(), "&")
}

// dumpCurlHeaders dumps headers to curl format
func dumpCurlHeaders(req *http.Request) *[][2]string {
	headers := [][2]string{}
	for k, vs := range req.Header {
		for _, v := range vs {
			headers = append(headers, [2]string{k, v})
		}
	}
	n := len(headers)
	for i := 0; i < n; i++ {
		for j := n - 1; j > i; j-- {
			jj := j - 1
			h1, h2 := headers[j], headers[jj]
			if h1[0] < h2[0] {
				headers[jj], headers[j] = headers[j], headers[jj]
			}
		}
	}
	return &headers
}

var regexCmdQuote = regexp.MustCompile(`[^\w@%+=:,./-]`)

// cmdQuote method to escape arbitrary strings for a safe use as
// command line arguments in the most common POSIX shells.
//
// The original Python package which this work was inspired by can be found
// at https://pypi.python.org/pypi/shellescape.
func cmdQuote(s string) string {
	if len(s) == 0 {
		return "''"
	}

	if regexCmdQuote.MatchString(s) {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}

	return s
}