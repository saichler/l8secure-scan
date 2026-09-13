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

package dataimport

import (
	"encoding/json"
	"fmt"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"time"
)

// Post handles template export — fetches templates by ID and returns them as JSON.
func (h *TransferHandler) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8ImportTemplateExportRequest)
	if !ok {
		return object.NewError("invalid template export request type")
	}

	if len(req.TemplateIds) == 0 {
		return object.NewError("no template IDs specified")
	}

	var templates []*l8api.L8ImportTemplate
	for _, id := range req.TemplateIds {
		tmpl, err := fetchTemplate(id, vnic)
		if err != nil {
			return object.NewError("export failed for " + id + ": " + err.Error())
		}
		templates = append(templates, tmpl)
	}

	data, err := json.Marshal(templates)
	if err != nil {
		return object.NewError("export serialization failed: " + err.Error())
	}

	filename := fmt.Sprintf("import_templates_%s.json", time.Now().Format("2006-01-02"))

	return object.New(nil, &l8api.L8ImportTemplateExportResponse{
		ExportData:    string(data),
		Filename:      filename,
		TemplateCount: int32(len(templates)),
	})
}

// Put handles template import — deserializes JSON templates and stores them.
func (h *TransferHandler) Put(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8ImportTemplateImportRequest)
	if !ok {
		return object.NewError("invalid template import request type")
	}

	var templates []*l8api.L8ImportTemplate
	if err := json.Unmarshal([]byte(req.ImportData), &templates); err != nil {
		return object.NewError("import deserialization failed: " + err.Error())
	}

	handler, hasLocal := vnic.Resources().Services().ServiceHandler("ImprtTmpl", ServiceArea)
	if !hasLocal {
		return object.NewError("template service not available")
	}

	var importedNames, skippedNames []string

	for _, tmpl := range templates {
		// Check if template with same name exists
		existing, _ := fetchTemplate(tmpl.TemplateId, vnic)

		if existing != nil && !req.OverwriteExisting {
			skippedNames = append(skippedNames, tmpl.Name)
			continue
		}

		var resp ifs.IElements
		if existing != nil {
			resp = handler.Put(object.New(nil, tmpl), vnic)
		} else {
			resp = handler.Post(object.New(nil, tmpl), vnic)
		}

		if resp.Error() != nil {
			skippedNames = append(skippedNames, tmpl.Name+" (error: "+resp.Error().Error()+")")
			continue
		}
		importedNames = append(importedNames, tmpl.Name)
	}

	return object.New(nil, &l8api.L8ImportTemplateImportResponse{
		ImportedCount: int32(len(importedNames)),
		SkippedCount:  int32(len(skippedNames)),
		ImportedNames: importedNames,
		SkippedNames:  skippedNames,
	})
}
