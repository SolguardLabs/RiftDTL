package main

type TagSet map[string]string

func NewTagSet() TagSet {
	return TagSet{}
}

func (t TagSet) Set(key string, value string) {
	if key == "" {
		return
	}
	t[NormalizeID(key)] = value
}

func (t TagSet) Get(key string) string {
	return t[NormalizeID(key)]
}

func (t TagSet) Has(key string) bool {
	_, ok := t[NormalizeID(key)]
	return ok
}

func (t TagSet) Delete(key string) {
	delete(t, NormalizeID(key))
}

func (t TagSet) Clone() TagSet {
	out := NewTagSet()
	for key, value := range t {
		out[key] = value
	}
	return out
}

func (t TagSet) Merge(other TagSet) TagSet {
	out := t.Clone()
	for key, value := range other {
		out[key] = value
	}
	return out
}

func (t TagSet) Keys() []string {
	keys := make([]string, 0, len(t))
	for key := range t {
		keys = append(keys, key)
	}
	SortStrings(keys)
	return keys
}

func (t TagSet) Values() []string {
	keys := t.Keys()
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, t[key])
	}
	return values
}

func (t TagSet) Empty() bool {
	return len(t) == 0
}

func Tags(pairs ...string) TagSet {
	out := NewTagSet()
	for i := 0; i+1 < len(pairs); i += 2 {
		out.Set(pairs[i], pairs[i+1])
	}
	return out
}
