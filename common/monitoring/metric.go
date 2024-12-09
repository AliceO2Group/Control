package monitoring

type (
	TagsType   map[string]any
	ValuesType map[string]any
)

type Metric struct {
	Name      string     `json:"name"`
	Values    ValuesType `json:"values"`
	Tags      TagsType   `json:"tags,omitempty"`
	Timestamp int64      `json:"timestamp"`
}

func (metric *Metric) AddTag(tagName string, value any) {
	if metric.Tags == nil {
		metric.Tags = make(TagsType)
	}
	metric.Tags[tagName] = value
}

func (metric *Metric) AddValue(valueName string, value any) {
	if metric.Values == nil {
		metric.Values = make(ValuesType)
	}
	metric.Values[valueName] = value
}
