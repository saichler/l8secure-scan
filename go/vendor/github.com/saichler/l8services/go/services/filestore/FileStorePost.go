/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filestore

import (
	"crypto/sha256"
	"fmt"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"os"
	"path/filepath"
	"strconv"
)

func (this *FileStore) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8FileUploadRequest)
	if !ok {
		return object.NewError("invalid file upload request type")
	}

	if req.FileName == "" {
		return object.NewError("file name is required")
	}

	if len(req.FileData) == 0 {
		return object.NewError("file data is empty")
	}

	if len(req.FileData) > MaxFileSize {
		return object.NewError(fmt.Sprintf("file size %d exceeds maximum %d bytes", len(req.FileData), MaxFileSize))
	}

	hash := sha256.Sum256(req.FileData)
	checksum := fmt.Sprintf("%x", hash)

	docDir := req.DocumentId
	if docDir == "" {
		docDir = "general"
	}
	version := strconv.Itoa(int(req.Version))
	if req.Version == 0 {
		version = "1"
	}

	dirPath := filepath.Join(StorageRoot, docDir, version)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return object.NewError("failed to create storage directory: " + err.Error())
	}

	// Encrypt file data before writing to disk
	encrypted, err := vnic.Resources().Security().Encrypt(req.FileData)
	if err != nil {
		return object.NewError("failed to encrypt file: " + err.Error())
	}

	storagePath := filepath.Join(dirPath, req.FileName)
	if err := os.WriteFile(storagePath, []byte(encrypted), 0644); err != nil {
		return object.NewError("failed to write file: " + err.Error())
	}

	resp := &l8api.L8FileUploadResponse{
		StoragePath: storagePath,
		FileName:    req.FileName,
		FileSize:    int64(len(req.FileData)),
		MimeType:    detectMimeType(req.FileName),
		Checksum:    checksum,
	}
	return object.New(nil, resp)
}
