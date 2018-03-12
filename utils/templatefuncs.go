package utils

type TemplateFuncsData struct {
	funcs map[string]interface{}
	data interface{}
}

func (tf *TemplateFuncsData) Funcs() map[string]interface{} {
	return tf.funcs
}

func (tf *TemplateFuncsData) Data() interface{} {
	return tf.data
}

func NewTemplateFuncsData(funcs map[string]interface{}, data interface{}) *TemplateFuncsData {
	return &TemplateFuncsData{funcs, data}
}
