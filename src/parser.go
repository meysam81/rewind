package requestrewind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

func (r *Request) FromHttpRequest(httpReq *http.Request) (*Request, error) {
	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		return nil, err
	}
	defer httpReq.Body.Close()

	headerJSON, err := json.Marshal(httpReq.Header)
	if err != nil {
		return nil, err
	}

	var cookies []byte
	if len(httpReq.Cookies()) > 0 {
		cookies, err = json.Marshal(httpReq.Cookies())
		if err != nil {
			return nil, err
		}
	}

	params, err := json.Marshal(httpReq.URL.RawQuery)
	if err != nil {
		return nil, err
	}

	parsedRequest := &Request{
		ID:          ulid.Make(),
		Method:      httpReq.Method,
		Path:        httpReq.URL.Path,
		Headers:     headerJSON,
		QueryParams: params,
		RecordedAt:  time.Now(),
	}

	if len(body) > 0 {
		parsedRequest.Body = body
	}

	if len(cookies) > 0 {
		parsedRequest.Cookies = cookies
	}

	return parsedRequest, nil
}

func (r *Request) ToHttpRequest(remoteHost string) (httpReq *http.Request, err error) {
	remoteUrl := fmt.Sprintf("%s%s", remoteHost, r.Path)

	var params string
	if err := json.Unmarshal(r.QueryParams, &params); err != nil {
		return nil, err
	}
	if params != "" {
		remoteUrl = fmt.Sprintf("%s?%s", remoteUrl, params)
	}

	httpReq, err = http.NewRequest(r.Method, remoteUrl, nil)
	if err != nil {
		return nil, err
	}

	var headers map[string][]string
	if len(r.Headers) > 0 {
		if err := json.Unmarshal(r.Headers, &headers); err != nil {
			return nil, err
		}
	}
	for key, values := range headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	if len(r.Body) > 0 {
		httpReq.Body = io.NopCloser(bytes.NewReader(r.Body))
	}

	return
}
