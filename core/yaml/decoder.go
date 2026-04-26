package yaml

import (
	"fmt"
	"reflect"
	"strings"
)

// Unmarshal parses YAML data and stores the result in the value pointed to by v.
// It supports decoding into map[string]any, []any, primitive types, and structs
// with `yaml` struct tags.
func Unmarshal(data []byte, v any) error {
	node, err := Parse(data)
	if err != nil {
		return err
	}
	return decodeNode(node, reflect.ValueOf(v))
}

func decodeNode(node *Node, v reflect.Value) error {
	// v must be a pointer
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("yaml: cannot unmarshal into %v", v.Type())
	}
	return decodeInto(node, v.Elem())
}

func decodeInto(node *Node, v reflect.Value) error {
	// Handle interface{}/any
	if v.Kind() == reflect.Interface {
		result := nodeToInterface(node)
		if result == nil {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		v.Set(reflect.ValueOf(result))
		return nil
	}

	switch node.Type {
	case ScalarNode:
		return decodeScalar(node.Value, v)
	case MappingNode:
		return decodeMapping(node, v)
	case SequenceNode:
		return decodeSequence(node, v)
	}

	return nil
}

func decodeScalar(value string, v reflect.Value) error {
	interpreted := interpretScalar(value)

	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch iv := interpreted.(type) {
		case int:
			v.SetInt(int64(iv))
			return nil
		case float64:
			v.SetInt(int64(iv))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal %q into %v", value, v.Type())

	case reflect.Float32, reflect.Float64:
		switch fv := interpreted.(type) {
		case float64:
			v.SetFloat(fv)
			return nil
		case int:
			v.SetFloat(float64(fv))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal %q into %v", value, v.Type())

	case reflect.Bool:
		if bv, ok := interpreted.(bool); ok {
			v.SetBool(bv)
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal %q into %v", value, v.Type())

	case reflect.Ptr:
		if interpreted == nil {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decodeScalar(value, v.Elem())

	case reflect.Map:
		// scalar into map doesn't make sense
		return fmt.Errorf("yaml: cannot unmarshal scalar into %v", v.Type())

	case reflect.Slice:
		// scalar into slice doesn't make sense
		return fmt.Errorf("yaml: cannot unmarshal scalar into %v", v.Type())
	}

	return fmt.Errorf("yaml: cannot unmarshal scalar %q into %v", value, v.Type())
}

func decodeMapping(node *Node, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		for _, kv := range node.Pairs {
			keyVal := reflect.ValueOf(kv.Key)
			elemVal := reflect.New(v.Type().Elem()).Elem()
			if err := decodeInto(kv.Value, elemVal); err != nil {
				return err
			}
			v.SetMapIndex(keyVal, elemVal)
		}
		return nil

	case reflect.Struct:
		return decodeStruct(node, v)

	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decodeMapping(node, v.Elem())
	}

	return fmt.Errorf("yaml: cannot unmarshal mapping into %v", v.Type())
}

func decodeSequence(node *Node, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Slice:
		slice := reflect.MakeSlice(v.Type(), len(node.Children), len(node.Children))
		for i, child := range node.Children {
			if err := decodeInto(child, slice.Index(i)); err != nil {
				return err
			}
		}
		v.Set(slice)
		return nil

	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decodeSequence(node, v.Elem())
	}

	return fmt.Errorf("yaml: cannot unmarshal sequence into %v", v.Type())
}

func decodeStruct(node *Node, v reflect.Value) error {
	t := v.Type()

	// Build field map: yaml tag name -> field index
	fieldMap := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			// Use lowercase field name as fallback
			if tag == "-" {
				continue
			}
			fieldMap[strings.ToLower(field.Name)] = i
		} else {
			// Parse tag (handle "name,omitempty" etc.)
			name := tag
			if idx := strings.Index(tag, ","); idx >= 0 {
				name = tag[:idx]
			}
			fieldMap[name] = i
		}
	}

	for _, kv := range node.Pairs {
		idx, ok := fieldMap[kv.Key]
		if !ok {
			// Unknown field, skip
			continue
		}
		fieldVal := v.Field(idx)
		if !fieldVal.CanSet() {
			continue
		}
		if err := decodeInto(kv.Value, fieldVal); err != nil {
			return fmt.Errorf("yaml: error decoding field %q: %w", kv.Key, err)
		}
	}

	return nil
}
