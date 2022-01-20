package resource

import "github.com/ecletus/core"

func convertMapToMetaValues(context *core.Context, values map[string]interface{}, metaors []Metaor, root ...*MetaValue) (*MetaValues, error) {
	if len(root) == 0 || root[0] == nil {
		root = []*MetaValue{{}}
	}
	var (
		parent       = root[0]
		metaValues   = &MetaValues{}
		metaorMap    = make(map[string]Metaor)
		childMetaors []Metaor
		newMetaValue = func(key string, value interface{}) {
			var metaValue *MetaValue
			metaor := metaorMap[key]

			switch result := value.(type) {
			case map[string]interface{}:
				if metaor != nil {
					childMetaors = metaor.GetContextMetas(nil, context)
				}
				if children, err := convertMapToMetaValues(context, result, childMetaors, parent); err == nil {
					metaValue = &MetaValue{Parent: parent, Name: key, Meta: metaor, MetaValues: children}
				}
			case []interface{}:
				for idx, r := range result {
					if mr, ok := r.(map[string]interface{}); ok {
						if metaor != nil {
							childMetaors = metaor.GetContextMetas(nil, context)
						}
						if children, err := convertMapToMetaValues(context, mr, childMetaors, parent); err == nil {
							metaValue := &MetaValue{
								Parent:     parent,
								Name:       key,
								Meta:       metaor,
								MetaValues: children,
								Index:      idx,
							}
							metaValues.Values = append(metaValues.Values, metaValue)
						}
					} else {
						metaValue := &MetaValue{Parent: parent, Name: key, Value: result, Meta: metaor}
						metaValues.Values = append(metaValues.Values, metaValue)
						break
					}
				}
			default:
				metaValue = &MetaValue{Parent: parent, Name: key, Value: value, Meta: metaor}
			}

			if metaValue != nil {
				metaValues.Add(metaValue)
			}
		}
	)

	for _, metaor := range metaors {
		metaorMap[metaor.GetName()] = metaor
	}

	for key, value := range values {
		newMetaValue(key, value)
	}

	if err := metaValues.CheckRequirement(context, metaors...); err != nil {
		return nil, err
	}

	return metaValues, nil
}
