package metric

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"

	"github.com/influxdata/telegraf"
)

const MaxInt = int(^uint(0) >> 1)

type metric struct {
	name   string
	tags   []*telegraf.Tag
	fields []*telegraf.Field
	tm     time.Time

	tp        telegraf.ValueType
	aggregate bool
}

func New(
	name string,
	tags map[string]string,
	fields map[string]interface{},
	tm time.Time,
	tp ...telegraf.ValueType,
) (telegraf.Metric, error) {
	var vtype telegraf.ValueType
	if len(tp) > 0 {
		vtype = tp[0]
	} else {
		vtype = telegraf.Untyped
	}

	m := &metric{
		name:   name,
		tags:   nil,
		fields: nil,
		tm:     tm,
		tp:     vtype,
	}

	if len(tags) > 0 {
		m.tags = make([]*telegraf.Tag, 0, len(tags))
		for k, v := range tags {
			m.tags = append(m.tags,
				&telegraf.Tag{Key: k, Value: v})
		}
		sort.Slice(m.tags, func(i, j int) bool { return m.tags[i].Key < m.tags[j].Key })
	}

	m.fields = make([]*telegraf.Field, 0, len(fields))
	for k, v := range fields {
		v := convertField(v)
		if v == nil {
			continue
		}
		m.AddField(k, v)
	}

	return m, nil
}

func (m *metric) String() string {
	return fmt.Sprintf("%s %v %v %d", m.name, m.Tags(), m.Fields(), m.tm.UnixNano())
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Tags() map[string]string {
	tags := make(map[string]string, len(m.tags))
	for _, tag := range m.tags {
		tags[tag.Key] = tag.Value
	}
	return tags
}

func (m *metric) TagList() []*telegraf.Tag {
	return m.tags
}

func (m *metric) Fields() map[string]interface{} {
	fields := make(map[string]interface{}, len(m.fields))
	for _, field := range m.fields {
		fields[field.Key] = field.Value
	}

	return fields
}

func (m *metric) FieldList() []*telegraf.Field {
	return m.fields
}

func (m *metric) Time() time.Time {
	return m.tm
}

func (m *metric) Type() telegraf.ValueType {
	return m.tp
}

func (m *metric) SetName(name string) {
	m.name = name
}

func (m *metric) AddPrefix(prefix string) {
	m.name = prefix + m.name
}

func (m *metric) AddSuffix(suffix string) {
	m.name = m.name + suffix
}

func (m *metric) AddTag(key, value string) {
	for i, tag := range m.tags {
		if key > tag.Key {
			continue
		}

		if key == tag.Key {
			tag.Value = value
		}

		m.tags = append(m.tags, nil)
		copy(m.tags[i+1:], m.tags[i:])
		m.tags[i] = &telegraf.Tag{Key: key, Value: value}
		return
	}

	m.tags = append(m.tags, &telegraf.Tag{Key: key, Value: value})
}

func (m *metric) HasTag(key string) bool {
	for _, tag := range m.tags {
		if tag.Key == key {
			return true
		}
	}
	return false
}

func (m *metric) GetTag(key string) (string, bool) {
	for _, tag := range m.tags {
		if tag.Key == key {
			return tag.Value, true
		}
	}
	return "", false
}

func (m *metric) RemoveTag(key string) {
	for i, tag := range m.tags {
		if tag.Key == key {
			copy(m.tags[i:], m.tags[i+1:])
			m.tags[len(m.tags)-1] = nil
			m.tags = m.tags[:len(m.tags)-1]
			return
		}
	}
}

func (m *metric) AddField(key string, value interface{}) {
	for i, field := range m.fields {
		if key == field.Key {
			m.fields[i] = &telegraf.Field{Key: key, Value: convertField(value)}
		}
	}
	m.fields = append(m.fields, &telegraf.Field{Key: key, Value: convertField(value)})
}

func (m *metric) HasField(key string) bool {
	for _, field := range m.fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func (m *metric) GetField(key string) (interface{}, bool) {
	for _, field := range m.fields {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

func (m *metric) RemoveField(key string) {
	for i, field := range m.fields {
		if field.Key == key {
			copy(m.fields[i:], m.fields[i+1:])
			m.fields[len(m.fields)-1] = nil
			m.fields = m.fields[:len(m.fields)-1]
			return
		}
	}
}

func (m *metric) Copy() telegraf.Metric {
	m2 := &metric{
		name:      m.name,
		tags:      make([]*telegraf.Tag, len(m.tags)),
		fields:    make([]*telegraf.Field, len(m.fields)),
		tm:        m.tm,
		tp:        m.tp,
		aggregate: m.aggregate,
	}

	for i, tag := range m.tags {
		m2.tags[i] = tag
	}

	for i, field := range m.fields {
		m2.fields[i] = field
	}
	return m2
}

func (m *metric) SetAggregate(b bool) {
	m.aggregate = true
}

func (m *metric) IsAggregate() bool {
	return m.aggregate
}

func (m *metric) HashID() uint64 {
	h := fnv.New64a()
	h.Write([]byte(m.name))
	for _, tag := range m.tags {
		h.Write([]byte(tag.Key))
		h.Write([]byte(tag.Value))
	}
	return h.Sum64()
}

// Convert field to a supported type or nil if unconvertible
func convertField(v interface{}) interface{} {
	switch v := v.(type) {
	case float64:
		return v
	case int64:
		return v
	case string:
		if v == "" {
			return nil
		} else {
			return v
		}
	case bool:
		return v
	case int:
		return int64(v)
	case uint:
		if v <= uint(MaxInt) {
			return int64(v)
		} else {
			return int64(MaxInt)
		}
	case uint64:
		if v <= uint64(MaxInt) {
			return int64(v)
		} else {
			return int64(MaxInt)
		}
	case []byte:
		return string(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case float32:
		return float64(v)
	default:
		return nil
	}
}
