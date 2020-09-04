package resource

import (
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
)

var (
	isCurrentLevel = regexp.MustCompile("^[^.]+$")
	isNextLevel    = regexp.MustCompile(`^(([^.\[\]]+)(\[\d+\])?)(?:(\.[^.]+)+)$`)
)

// ConvertFormToMetaValues convert form to meta values
func ConvertFormDataToMetaValues(context *core.Context, form url.Values, multipartForm *multipart.Form, metaors []Metaor, prefix string, root ...*MetaValue) (*MetaValues, error) {
	if len(root) == 0 || root[0] == nil {
		root = []*MetaValue{{}}
	}

	var (
		sortedFormKeys []string
		errors         core.Errors

		metaValues         = &MetaValues{ByName: map[string]int{}}
		metaorsMap         = map[string]Metaor{}
		convertedNextLevel = map[string]bool{}
		nestedStructIndex  = map[string]int{}
		newMetaValue       = func(key string, value interface{}) (err error) {
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
							metaValue = &MetaValue{Parent: root[0], Name: name, Value: value, Meta: metaorsMap[name]}
						} else if len(files) > 0 {
							prev.Value = value
						}
					} else {
						metaValue = &MetaValue{Parent: root[0], Name: name, Value: value, Meta: metaorsMap[name]}
					}
				} else if matches := isNextLevel.FindStringSubmatch(key); len(matches) > 0 {
					name := matches[1]
					if _, ok := convertedNextLevel[name]; !ok {
						var (
							hasParent bool
							parent    = metaValues.Get(name)
							metaors   []Metaor
						)
						convertedNextLevel[name] = true
						metaor := metaorsMap[matches[2]]
						if metaor != nil {
							metaors = metaor.GetContextMetas(nil, context)
						}

						if parent != nil {
							hasParent = true
						} else {
							parent = &MetaValue{Name: matches[2], Meta: metaor, Parent: root[0]}
						}

						if children, err := ConvertFormDataToMetaValues(context, form, multipartForm, metaors, prefix+name+".", parent); err != nil {
							return err
						} else {
							nestedName := prefix + matches[2]
							if _, ok := nestedStructIndex[nestedName]; ok {
								nestedStructIndex[nestedName]++
							} else {
								nestedStructIndex[nestedName] = 0
							}
							if parent.MetaValues == nil {
								parent.MetaValues = children
							} else {
								parent.MetaValues.Values = append(parent.MetaValues.Values, children.Values...)
							}
							if !hasParent {
								parent.Index = nestedStructIndex[nestedName]
								metaValue = parent
							}
						}
					}
				}

				if metaValue != nil {
					metaValues.ByName[metaValue.Name] = len(metaValues.Values)
					metaValues.Values = append(metaValues.Values, metaValue)
				}
			}
			return nil
		}
	)

	for _, metaor := range metaors {
		metaorsMap[metaor.GetName()] = metaor
	}

	for key := range form {
		sortedFormKeys = append(sortedFormKeys, key)
	}

	utils.SortFormKeys(sortedFormKeys)

	for _, key := range sortedFormKeys {
		if err := newMetaValue(key, form[key]); err != nil {
			errors.AddError(err)
		}
	}

	errors.AddError(metaValues.CheckRequirement(context, metaors...))

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
	if errors.HasError() {
		return metaValues, errors
	}
	return metaValues, nil
}

// ConvertFormToMetaValues convert form to meta values
func ConvertFormToMetaValues(context *core.Context, request *http.Request, metaors []Metaor, prefix string, root ...*MetaValue) (*MetaValues, error) {
	return ConvertFormDataToMetaValues(context, request.Form, request.MultipartForm, metaors, prefix, root...)
}
