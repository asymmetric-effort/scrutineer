package connector

// GetString extracts a string value from Result.Data by key.
func (r *Result) GetString(key string) (string, bool) {
	if r.Data == nil {
		return "", false
	}
	v, ok := r.Data[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt extracts an int value from Result.Data by key.
// Supports int, int64, and float64 (truncated) source types.
func (r *Result) GetInt(key string) (int, bool) {
	if r.Data == nil {
		return 0, false
	}
	v, ok := r.Data[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// GetFloat extracts a float64 value from Result.Data by key.
// Supports float64 and int source types.
func (r *Result) GetFloat(key string) (float64, bool) {
	if r.Data == nil {
		return 0, false
	}
	v, ok := r.Data[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

// GetBool extracts a bool value from Result.Data by key.
func (r *Result) GetBool(key string) (bool, bool) {
	if r.Data == nil {
		return false, false
	}
	v, ok := r.Data[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// GetMap extracts a map[string]any value from Result.Data by key.
func (r *Result) GetMap(key string) (map[string]any, bool) {
	if r.Data == nil {
		return nil, false
	}
	v, ok := r.Data[key]
	if !ok {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}
