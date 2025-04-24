package hub

type ExtraData map[string]any

func (e ExtraData) Get(key string) any {
	if v, ok := e[key]; ok {
		return v
	}
	return nil
}

func (e ExtraData) GetString(key string) string {
	if v, ok := e[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (e ExtraData) GetInt(key string) int {
	if v, ok := e[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

func (e ExtraData) Set(key string, value any) {
	e[key] = value
}
func (e ExtraData) Delete(key string) {
	delete(e, key)
}
func (e ExtraData) Clear() {
	for k := range e {
		delete(e, k)
	}
}
func (e ExtraData) Len() int {
	return len(e)
}
func (e ExtraData) IsEmpty() bool {
	return len(e) == 0
}
