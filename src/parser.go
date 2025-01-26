package rewind

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
)

func (r *RequestWriteDB) FromHttpRequest(httpReq *http.Request) (parsedRequest *RequestWriteDB, err error) {
	body, err := io.ReadAll(httpReq.Body)
	if err != nil {
		return nil, err
	}
	defer httpReq.Body.Close()

	headers, err := json.Marshal(httpReq.Header)
	if err != nil {
		return nil, err
	}

	cookies, err := json.Marshal(httpReq.Cookies())
	if err != nil {
		return nil, err
	}

	params, err := json.Marshal(httpReq.URL.Query())
	if err != nil {
		return nil, err
	}

	parsedRequest = &RequestWriteDB{
		ID:          ulid.Make(),
		Method:      httpReq.Method,
		Path:        httpReq.URL.Path,
		Headers:     headers,
		Cookies:     cookies,
		Body:        body,
		QueryParams: params,
		RecordedAt:  time.Now(),
	}

	return parsedRequest, nil
}

func (r *RequestWriteDB) ToHttpRequest(remoteHost string) (httpReq *http.Request, err error) {
	return
}
