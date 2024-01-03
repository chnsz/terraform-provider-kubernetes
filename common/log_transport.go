package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

const (
	LogReqMsg = `[DEBUG] %s API Request Details:
----[ REQUEST ]---------------------------------------
%s
------------------------------------------------------`

	LogRespMsg = `[DEBUG] %s API Response Details:
----[ RESPONSE ]--------------------------------------
%s
------------------------------------------------------`
)

type LogTransport struct {
	name      string
	transport http.RoundTripper
}

func (t *LogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if logging.IsDebugOrHigher() {
		reqData, err := httputil.DumpRequest(req, true)
		if err == nil {
			log.Printf(LogReqMsg, t.name, prettyPrintJsonLines(reqData))
		} else {
			log.Printf("[ERROR] %s API Request error: %#v", t.name, err)
		}
	}

	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if logging.IsDebugOrHigher() {
		respData, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Printf(LogRespMsg, t.name, prettyPrintJsonLines(respData))
		} else {
			log.Printf("[ERROR] %s API Response error: %#v", t.name, err)
		}
	}

	return resp, nil
}

func NewLogTransport(name string, t http.RoundTripper) *LogTransport {
	return &LogTransport{name, t}
}

// prettyPrintJsonLines iterates through a []byte line-by-line,
// transforming any lines that are complete json into pretty-printed json.
func prettyPrintJsonLines(b []byte) string {
	parts := strings.Split(string(b), "\n")
	for i, p := range parts {
		if b := []byte(p); json.Valid(b) {
			var out bytes.Buffer
			_ = json.Indent(&out, b, "", " ") // already checked for validity
			parts[i] = out.String()
		}
	}

	for i := range parts {
		parts[i] = removeSensitive(parts[i])
	}

	return strings.Join(parts, "\n")
}

var sensitiveKeywords = []string{"Authorization", "X-Security-Token", "stringData", "data", "secretName"}

func removeSensitive(s string) string {
	reg1 := regexp.MustCompile(fmt.Sprintf(`("%s"):\s*"(.*?)"`, strings.Join(sensitiveKeywords, `"|"`)))
	s = reg1.ReplaceAllString(s, `$1: "******"`)

	reg2 := regexp.MustCompile(fmt.Sprintf(`(%s):\s*(.*?)$`, strings.Join(sensitiveKeywords, `|`)))
	s = reg2.ReplaceAllString(s, `$1: ******`)

	return s
}
