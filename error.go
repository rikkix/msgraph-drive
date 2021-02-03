package drive

import (
	"encoding/json"
	"fmt"
)

type ReqError struct {
	URL        string `json:"-"`
	StatusCode int    `json:"-"`
	Err        struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		InnerError struct {
			Date            string `json:"date"`
			RequestID       string `json:"request-id"`
			ClientRequestID string `json:"client-request-id"`
		} `json:"innerError"`
	} `json:"error"`
	notValid bool   `json:"-"`
	Raw      string `json:"-"`
}

func NewErr(statusCode int, body []byte) error {
	e := &ReqError{
		StatusCode: statusCode,
		Raw:        string(body),
	}
	err := json.Unmarshal(body, e)
	if err != nil {
		e.notValid = true
		return e
	}
	return e
}

func (re *ReqError) String() string {
	if re.notValid {
		return fmt.Sprintf("StatusCode is not OK: %v. Body: %v", re.StatusCode, re.Raw)

	}
	return fmt.Sprintf("StatusCode is not OK: %v(%v).", re.StatusCode, re.Err.Code)
}

func (re *ReqError) Error() string {
	return re.String()
}
