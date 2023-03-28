/*
 * Copyright 2023 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var ErrNotFound = errors.New("not found")
var ErrAccessDenied = errors.New("access denied")
var ErrBadRequest = errors.New("bad request")
var ErrInvalidAuth = errors.New("invalid auth token")

func GetErrCode(err error) (code int) {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, ErrBadRequest) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrAccessDenied) {
		return http.StatusForbidden
	}
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrInvalidAuth) {
		return http.StatusUnauthorized
	}
	return http.StatusInternalServerError
}

func GetErrFromCode(code int, msg string) error {
	if code < 300 {
		return nil
	}

	if msg == "" {
		msg = "received status code " + strconv.Itoa(code)
	}

	//clean message
	cleanedMessage := msg
	cleanedMessage = strings.TrimPrefix(msg, ErrBadRequest.Error())
	cleanedMessage = strings.TrimPrefix(msg, ErrAccessDenied.Error())
	cleanedMessage = strings.TrimPrefix(msg, ErrBadRequest.Error())
	cleanedMessage = strings.TrimPrefix(msg, ErrInvalidAuth.Error())
	cleanedMessage = strings.TrimSpace(msg)
	cleanedMessage = strings.TrimPrefix(msg, ":")
	cleanedMessage = strings.TrimSpace(msg)
	if cleanedMessage == "" {
		cleanedMessage = msg
	}

	switch code {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %v", ErrBadRequest, cleanedMessage)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %v", ErrAccessDenied, cleanedMessage)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %v", ErrNotFound, cleanedMessage)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %v", ErrInvalidAuth, cleanedMessage)
	default:
		return errors.New(msg)
	}
}
