package resource

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/aghape/core"
	"github.com/aghape/core/utils"
)

func convertMapToMetaValues(values map[string]interface{}, metaors []Metaor) (*MetaValues, error) {
	metaValues := &MetaValues{}
	metaorMap := make(map[string]Metaor)
	for _, metaor := range metaors {
		metaorMap[metaor.GetName()] = metaor
	}

	for key, value := range values {
		var metaValue *MetaValue
		metaor := metaorMap[key]
		var childMeta []Metaor
		if metaor != nil {
			childMeta = metaor.GetMetas()
		}

		switch result := value.(type) {
		case map[string]interface{}:
			if children, err := convertMapToMetaValues(result, childMeta); err == nil {
				metaValue = &MetaValue{Name: key, Meta: metaor, MetaValues: children}
			}
		case []interface{}:
			for idx, r := range result {
				if mr, ok := r.(map[string]interface{}); ok {
					if children, err := convertMapToMetaValues(mr, childMeta); err == nil {
						metaValue := &MetaValue{Name: key, Meta: metaor, MetaValues: children, Index: idx}
						metaValues.Values = append(metaValues.Values, metaValue)
					}
				} else {
					metaValue := &MetaValue{Name: key, Value: result, Meta: metaor}
					metaValues.Values = append(metaValues.Values, metaValue)
					break
				}
			}
		default:
			metaValue = &MetaValue{Name: key, Value: value, Meta: metaor}
		}

		if metaValue != nil {
			metaValues.Values = append(metaValues.Values, metaValue)
		}
	}
	return metaValues, nil
}

// ConvertJSONToMetaValues convert json to meta values
func ConvertJSONToMetaValues(reader io.Reader, metaors []Metaor) (*MetaValues, error) {
	var (
		err     error
		values  = map[string]interface{}{}
		decoder = json.NewDecoder(reader)
	)

	if err = decoder.Decode(&values); err == nil {
		return convertMapToMetaValues(values, metaors)
	}
	return nil, err
}

var (
	isCurrentLevel = regexp.MustCompile("^[^.]+$")
	isNextLevel    = regexp.MustCompile(`^(([^.\[\]]+)(\[\d+\])?)(?:(\.[^.]+)+)$`)
)

// ConvertFormToMetaValues convert form to meta values
func ConvertFormDataToMetaValues(context *core.Context, form url.Values, multipartForm *multipart.Form, metaors []Metaor, prefix string) (*MetaValues, error) {
	metaValues := &MetaValues{}
	metaorsMap := map[string]Metaor{}
	convertedNextLevel := map[string]bool{}
	nestedStructIndex := map[string]int{}
	for _, metaor := range metaors {
		metaorsMap[metaor.GetName()] = metaor
	}

	newMetaValue := func(key string, value interface{}) error {
		if strings.HasPrefix(key, prefix) {
			var metaValue *MetaValue
			key = strings.TrimPrefix(key, prefix)

			if matches := isCurrentLevel.FindStringSubmatch(key); len(matches) > 0 {
				name := matches[0]
				// skip if has previous meta with same name and
				// this is file and has not be set
				if values, ok := value.([]*multipart.FileHeader); ok {
					var files []*multipart.FileHeader
					for _, f := range values {
						if f.Filename != "" {
							files = append(files, f)
						}
					}

					prev := metaValues.Get(name)
					if prev == nil {
						metaValue = &MetaValue{Name: name, Value: value, Meta: metaorsMap[name]}
					} else if len(files) > 0 {
						prev.Value = value
					}
				} else {
					metaValue = &MetaValue{Name: name, Value: value, Meta: metaorsMap[name]}
				}
			} else if matches := isNextLevel.FindStringSubmatch(key); len(matches) > 0 {
				name := matches[1]
				if _, ok := convertedNextLevel[name]; !ok {
					var metaors []Metaor
					convertedNextLevel[name] = true
					metaor := metaorsMap[matches[2]]
					if metaor != nil {
						metaors = metaor.GetContextMetas(nil, context)
					}

					if children, err := ConvertFormDataToMetaValues(context, form, multipartForm, metaors, prefix+name+"."); err == nil {
						nestedName := prefix + matches[2]
						if _, ok := nestedStructIndex[nestedName]; ok {
							nestedStructIndex[nestedName]++
						} else {
							nestedStructIndex[nestedName] = 0
						}
						metaValue = &MetaValue{Name: matches[2], Meta: metaor, MetaValues: children, Index: nestedStructIndex[nestedName]}
					}
				}
			}

			if metaValue != nil {
				metaValues.Values = append(metaValues.Values, metaValue)
			}
		}
		return nil
	}

	var sortedFormKeys []string
	for key := range form {
		sortedFormKeys = append(sortedFormKeys, key)
	}

	utils.SortFormKeys(sortedFormKeys)

	for _, key := range sortedFormKeys {
		newMetaValue(key, form[key])
	}

	if multipartForm != nil {
		sortedFormKeys = []string{}
		for key := range multipartForm.File {
			sortedFormKeys = append(sortedFormKeys, key)
		}
		utils.SortFormKeys(sortedFormKeys)

		for _, key := range sortedFormKeys {
			newMetaValue(key, multipartForm.File[key])
		}
	}
	return metaValues, nil
}

// ConvertFormToMetaValues convert form to meta values
func ConvertFormToMetaValues(context *core.Context, request *http.Request, metaors []Metaor, prefix string) (*MetaValues, error) {
	return ConvertFormDataToMetaValues(context, request.Form, request.MultipartForm, metaors, prefix)
}

// Decode decode context to result according to resource definition
func Decode(context *core.Context, result interface{}, res Resourcer) error {
	var errors core.Errors
	var err error
	var metaValues *MetaValues

	if parent := res.GetParentResource(); parent != nil {
		parentId := context.ParentResourceID[parent.GetPathLevel()]
		value := reflect.Indirect(reflect.ValueOf(result))
		fieldName := res.GetParentFieldName()
		f := value.FieldByName(fieldName)
		f.Set(reflect.ValueOf(parentId))
	}

	metaors := res.GetMetas([]string{})

	if strings.Contains(context.Request.Header.Get("Content-Type"), "json") {
		metaValues, err = ConvertJSONToMetaValues(context.Request.Body, metaors)
		context.Request.Body.Close()
	} else {
		metaValues, err = ConvertFormToMetaValues(context, context.Request, metaors, "QorResource.")
	}

	errors.AddError(err)
	processor := DecodeToResource(res, result, metaValues, context)
	err = processor.Start()
	errors.AddError(err)

	return errors
}
