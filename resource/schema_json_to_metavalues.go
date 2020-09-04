package resource

import (
	"encoding/json"
	"io"

	"github.com/ecletus/core"
)

// ConvertJSONToMetaValues convert json to meta values
func ConvertJSONToMetaValues(context *core.Context, reader io.Reader, metaors []Metaor) (*MetaValues, error) {
	var (
		err     error
		values  = map[string]interface{}{}
		decoder = json.NewDecoder(reader)
	)

	if err = decoder.Decode(&values); err == nil {
		return convertMapToMetaValues(context, values, metaors)
	}
	return nil, err
}
