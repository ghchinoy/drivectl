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

package drive

import (
	"fmt"

	"google.golang.org/api/drive/v3"
)

// ListRevisions retrieves a list of revisions for a given file ID.
func ListRevisions(srv *drive.Service, fileId string) ([]*drive.Revision, error) {
	call := srv.Revisions.List(fileId).Fields("revisions(id,modifiedTime,lastModifyingUser(displayName,emailAddress),size,originalFilename,mimeType)")
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list revisions: %w", err)
	}
	return resp.Revisions, nil
}
