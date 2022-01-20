package resource

import (
	"github.com/ecletus/core"
)

func FormTreeWalkMetaValues(t *FormTree, context *core.Context, metaorsMap map[string]Metaor, this *MetaValue) (err error) {
	var errors core.Errors
	this.Index = t.Index

	if t.Children.Map == nil {
		this.Value = t.Value
		this.Parent.MetaValues.Add(this)
		return nil
	}

	var disabled = t.Children.GetStringValue("@enabled") == "false"
	if disabled {
		this.MetaValues = &MetaValues{Disabled: true}
		this.Parent.MetaValues.Add(this)
		return nil
	}

	var (
		metaor        = metaorsMap[t.Key]
		subMetaorsMap map[string]Metaor
		metaors       []Metaor
		addChild      = func(child *FormTree, childMeta Metaor) {
			mv := &MetaValue{
				Parent: this,
				Name:   child.Key,
				Meta:   childMeta,
			}

			if childMeta == nil && len(child.Children.Keys) > 0 {
				res := this.Meta.GetResource()
				if res != nil {
					for _, m := range res.GetMetas([]string{mv.Name}) {
						if m.GetName() == child.Key {
							mv.Meta = m
							subMetaorsMap[mv.Name] = m
							goto ok
						}
					}
				}
				return
			}
		ok:
			if err := FormTreeWalkMetaValues(child, context, subMetaorsMap, mv); err != nil {
				errors.AddError(err)
			}
		}
	)

	if this.Parent == nil || this.Index != -1 {
		subMetaorsMap = metaorsMap
		if this.Index != -1 {
			this.Parent.MetaValues.Add(this)
		}
	} else {
		if metaor != nil {
			metaors = metaor.GetContextMetas(nil, context)
		} else if this.Parent != nil && !this.Meta.CanCollection() {
			metaors = nil
		}

		subMetaorsMap = map[string]Metaor{}

		for _, metaor := range metaors {
			subMetaorsMap[metaor.GetName()] = metaor
		}
		this.Parent.MetaValues.Add(this)
	}

	if len(metaors) == 0 {
		for _, m := range subMetaorsMap {
			metaors = append(metaors, m)
		}
	}

	this.MetaValues = &MetaValues{}

	if t.Children.NextIndex > 0 {
		for _, child := range t.Children.Slice {
			addChild(child, this.Meta)
		}
	} else {
		for _, key := range t.Children.Keys {
			addChild(t.Children.Map[key], subMetaorsMap[key])
		}
		errors.AddError(this.MetaValues.CheckRequirement(context, metaors...))
	}

	if errors.HasError() {
		return errors
	}
	return nil
}
