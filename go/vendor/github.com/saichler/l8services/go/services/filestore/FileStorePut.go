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
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"os"
	"path/filepath"
	"strings"
)

func (this *FileStore) Put(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8FileDownloadRequest)
	if !ok {
		return object.NewError("invalid file download request type")
	}

	if req.StoragePath == "" {
		return object.NewError("storage path is required")
	}

	cleanPath := filepath.Clean(req.StoragePath)
	if !strings.HasPrefix(cleanPath, StorageRoot) {
		return object.NewError("access denied: path outside storage root")
	}

	encryptedData, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return object.NewError("file not found: " + req.StoragePath)
		}
		return object.NewError("failed to read file: " + err.Error())
	}

	// Decrypt file data
	data, err := vnic.Resources().Security().Decrypt(string(encryptedData))
	if err != nil {
		return object.NewError("failed to decrypt file: " + err.Error())
	}

	fileName := filepath.Base(cleanPath)

	resp := &l8api.L8FileDownloadResponse{
		FileData: data,
		FileName: fileName,
		MimeType: detectMimeType(fileName),
		FileSize: int64(len(data)),
	}
	return object.New(nil, resp)
}
