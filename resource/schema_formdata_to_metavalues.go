package resource

import (
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
)

// ConvertFormToMetaValues convert form to meta values
func ConvertFormDataToMetaValues(context *core.Context, form url.Values, multipartForm *multipart.Form, metaors []Metaor, prefix string, root ...*MetaValue) (_ *MetaValues, err error) {
	if len(root) == 0 || root[0] == nil {
		root = []*MetaValue{{}}
	}
	var (
		sortedFormKeys []string
		tree           = NewFormTree()
		metaorsMap     = map[string]Metaor{}
	)

	if prefix != "" {
		for key := range form {
			if strings.HasPrefix(key, prefix) {
				sortedFormKeys = append(sortedFormKeys, key)
			}
		}
		if multipartForm != nil {
			for key := range multipartForm.File {
				if strings.HasPrefix(key, prefix) {
					sortedFormKeys = append(sortedFormKeys, key)
				}
			}
		}
	} else {
		for key := range form {
			sortedFormKeys = append(sortedFormKeys, key)
		}
		if multipartForm != nil {
			for key := range multipartForm.File {
				if strings.HasPrefix(key, prefix) {
					sortedFormKeys = append(sortedFormKeys, key)
				}
			}
		}
	}

	utils.SortFormKeys(sortedFormKeys)

	if multipartForm == nil {
		for _, key := range sortedFormKeys {
			tree.Of(strings.TrimPrefix(key, prefix)).Value = form[key]
		}
	} else {
		last := len(sortedFormKeys) - 1
		for i, key := range sortedFormKeys {
			if i < last && strings.HasPrefix(sortedFormKeys[i+1], key+".") {
				key += ".id"
			}
			el := tree.Of(strings.TrimPrefix(key, prefix))
			el.Value = multipartForm.Value[key]
			var files []*multipart.FileHeader
			for _, f := range multipartForm.File[key] {
				if f.Filename != "" {
					files = append(files, f)
				}
			}
			if len(files) > 0 {
				el.Value = files
			}
		}
	}
	for _, metaor := range metaors {
		metaorsMap[metaor.GetName()] = metaor
	}

	if root[0].MetaValues == nil {
		root[0].MetaValues = &MetaValues{}
	}

	root[0].Index = -1
	tree.Index = -1

	if tree.Children.Map != nil {
		err = FormTreeWalkMetaValues(tree, context, metaorsMap, root[0])
		if err != nil {
			return nil, err
		}
	}
	return root[0].MetaValues, nil
}

// ConvertFormToMetaValues convert form to meta values
func ConvertFormToMetaValues(context *core.Context, request *http.Request, metaors []Metaor, prefix string, root ...*MetaValue) (*MetaValues, error) {
	return ConvertFormDataToMetaValues(context, request.Form, request.MultipartForm, metaors, prefix, root...)
}
