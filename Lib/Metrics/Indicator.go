package Metrics

import (
	"github.com/castai/promwrite"
	"slices"
	"time"
)

type Indicator struct {
	Name   string
	Labels map[string]string
	Value  float64
	Time   time.Time
}

func NewIndicator(name string) *Indicator {
	return &Indicator{
		Name:   name,
		Labels: make(map[string]string),
	}
}

func NewIndicatorFull(name string, time time.Time, value float64, labels map[string]string, extraLabels map[string]string) *Indicator {
	var indicator = &Indicator{
		Name:  name,
		Time:  time,
		Value: value,
	}

	indicator.SetLabels(labels)
	indicator.AddLabels(extraLabels)

	return indicator
}

func (indicator *Indicator) SetLabels(labels map[string]string) *Indicator {
	indicator.Labels = labels
	return indicator
}

func (indicator *Indicator) AddLabels(labels map[string]string) *Indicator {
	for name, value := range labels {
		indicator.Labels[name] = value
	}
	return indicator
}

func (indicator *Indicator) SetValue(value float64) *Indicator {
	indicator.Value = value
	return indicator
}

func sortMapByKey(m *map[string]string) map[string]string {
	keys := make([]string, 0, len(*m))
	for k := range *m {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	var sortedMap = make(map[string]string)
	for _, k := range keys {
		sortedMap[k] = (*m)[k]
	}

	return sortedMap
}

func (indicator *Indicator) ToTimeSeries() promwrite.TimeSeries {
	var labels = make([]promwrite.Label, len(indicator.Labels)+1)

	labels[0] = promwrite.Label{
		Name:  "__name__",
		Value: indicator.Name,
	}

	var i = 1

	indicator.Labels = sortMapByKey(&indicator.Labels)

	for key, value := range indicator.Labels {
		labels[i] = promwrite.Label{
			Name:  key,
			Value: value,
		}
		i++
	}

	return promwrite.TimeSeries{
		Labels: labels,
		Sample: promwrite.Sample{
			Time:  indicator.Time,
			Value: indicator.Value,
		},
	}
}
