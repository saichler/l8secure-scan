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
	"encoding/base64"
	"fmt"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

func (h *ExecuteHandler) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8ImportExecuteRequest)
	if !ok {
		return object.NewError("invalid import execute request type")
	}

	// Fetch the template via the template CRUD service
	tmpl, err := fetchTemplate(req.TemplateId, vnic)
	if err != nil {
		return object.NewError("template load failed: " + err.Error())
	}

	// Decode file data
	fileData, err := base64.StdEncoding.DecodeString(req.FileData)
	if err != nil {
		return object.NewError("file decode failed: " + err.Error())
	}

	// Parse file
	parsed, err := ParseFile(fileData, tmpl.SourceFormat, req.FileName)
	if err != nil {
		return object.NewError("file parse failed: " + err.Error())
	}

	// Get target node for record building
	node, found := vnic.Resources().Introspector().NodeByTypeName(tmpl.TargetModelType)
	if !found {
		return object.NewError("unknown model type: " + tmpl.TargetModelType)
	}

	// Get target service handler
	handler, hasLocal := vnic.Resources().Services().ServiceHandler(
		tmpl.TargetServiceName, byte(tmpl.TargetServiceArea))
	if !hasLocal {
		return object.NewError("service not found: " + tmpl.TargetServiceName)
	}

	var rowErrors []*l8api.L8ImportRowError
	imported := int32(0)

	for _, row := range parsed.Rows {
		mapped := MapRow(row, tmpl.ColumnMappings)
		ApplyConcatenation(mapped, tmpl.ValueTransforms)

		record, buildErrs := BuildRecord(mapped, node,
			vnic.Resources().Registry(), tmpl.ValueTransforms, tmpl.DefaultValues)
		if len(buildErrs) > 0 {
			for _, e := range buildErrs {
				rowErrors = append(rowErrors, &l8api.L8ImportRowError{
					RowNumber:    int32(row.RowNumber),
					ErrorMessage: e.Error(),
				})
			}
			continue
		}

		postElems := object.New(nil, record)
		resp := handler.Post(postElems, vnic)
		if resp.Error() != nil {
			rowErrors = append(rowErrors, &l8api.L8ImportRowError{
				RowNumber:    int32(row.RowNumber),
				ErrorMessage: resp.Error().Error(),
			})
			continue
		}
		imported++
	}

	return object.New(nil, &l8api.L8ImportExecuteResponse{
		TotalRows:    int32(len(parsed.Rows)),
		ImportedRows: imported,
		FailedRows:   int32(len(rowErrors)),
		RowErrors:    rowErrors,
	})
}

// fetchTemplate retrieves an import template by ID via the template CRUD service.
func fetchTemplate(templateId string, vnic ifs.IVNic) (*l8api.L8ImportTemplate, error) {
	handler, hasLocal := vnic.Resources().Services().ServiceHandler("ImprtTmpl", ServiceArea)
	if !hasLocal {
		return nil, fmt.Errorf("template service not available")
	}

	filter := &l8api.L8ImportTemplate{TemplateId: templateId}
	resp := handler.Get(object.New(nil, filter), vnic)
	if resp.Error() != nil {
		return nil, resp.Error()
	}

	tmpl, ok := resp.Element().(*l8api.L8ImportTemplate)
	if !ok {
		return nil, fmt.Errorf("template not found: %s", templateId)
	}
	return tmpl, nil
}
