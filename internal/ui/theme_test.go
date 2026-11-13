// Copyright 2025 Google LLC
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

package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorWithHint(t *testing.T) {
	baseErr := errors.New("connection refused")
	hint := "check network settings"

	gotErr := ErrorWithHint(baseErr, hint)
	if !errors.Is(gotErr, baseErr) {
		t.Errorf("ErrorWithHint(%v, %q) = %v, want error wrapping %v", baseErr, hint, gotErr, baseErr)
	}
	if !strings.Contains(gotErr.Error(), hint) {
		t.Errorf("ErrorWithHint(%v, %q) = %q, want string containing %q", baseErr, hint, gotErr.Error(), hint)
	}
}

func TestHandleError_Nil(t *testing.T) {
	HandleError(nil)
}

func TestPrintSuccess(t *testing.T) {
	PrintSuccess("completed %d items", 3)
}
