// Copyright (c) 2018 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/palantir/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/palantir/conjure-go-runtime/conjure-go-contract/errors"
)

func TestErrorFromResponse(t *testing.T) {
	tests := map[string]errors.SerializableError{
		"default timeout": {
			ErrorCode:       errors.DefaultTimeout.Code(),
			ErrorName:       errors.DefaultTimeout.Name(),
			ErrorInstanceID: uuid.NewUUID(),
			Parameters:      json.RawMessage(`{"ttl":"10s"}`),
		},
		"custom timeout": {
			ErrorCode:       errors.DefaultTimeout.Code(),
			ErrorName:       "MyApplication:Timeout",
			ErrorInstanceID: uuid.NewUUID(),
			Parameters:      json.RawMessage(`{"ttl":"10s"}`),
		},
		"custom not found": {
			ErrorCode:       errors.NotFound,
			ErrorName:       "MyApplication:MissingData",
			ErrorInstanceID: uuid.NewUUID(),
		},
		"custom client": {
			ErrorCode:       errors.CustomClient,
			ErrorName:       "MyApplication:CustomClientError",
			ErrorInstanceID: uuid.NewUUID(),
		},
		"custom server": {
			ErrorCode:       errors.CustomServer,
			ErrorName:       "MyApplication:CustomServerError",
			ErrorInstanceID: uuid.NewUUID(),
		},
	}

	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			marshalledError, err := json.Marshal(expected)
			require.NoError(t, err)

			response := &http.Response{
				Status:     http.StatusText(expected.ErrorCode.StatusCode()),
				StatusCode: expected.ErrorCode.StatusCode(),
				Header: http.Header{
					"Content-Type": []string{"application/json; charset=utf-8"},
				},
				Body: ioutil.NopCloser(bytes.NewBuffer(marshalledError)),
			}

			actual, err := errors.ErrorFromResponse(response)
			assert.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestWriteErrorResponse_ValidateJSON(t *testing.T) {
	testSerializableError := errors.SerializableError{
		ErrorCode:       errors.Timeout,
		ErrorName:       "MyApplication:Timeout",
		ErrorInstanceID: uuid.NewUUID(),
		Parameters: json.RawMessage(`{
    "metadata": {
      "keyB": 4
    }
  }`),
	}

	testErrorJSON := fmt.Sprintf(`{
  "errorCode": "TIMEOUT",
  "errorName": "MyApplication:Timeout",
  "errorInstanceId": "%s",
  "parameters": {
    "metadata": {
      "keyB": 4
    }
  }
}`, testSerializableError.ErrorInstanceID)

	recorder := httptest.NewRecorder()
	errors.WriteErrorResponse(recorder, testSerializableError)
	response := recorder.Result()

	assert.Equal(t, "application/json; charset=utf-8", response.Header.Get("Content-Type"))
	body, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)

	var buffer bytes.Buffer
	require.NoError(t, json.Indent(&buffer, body, "", "  "))
	assert.Equal(t, testErrorJSON, buffer.String())
}
