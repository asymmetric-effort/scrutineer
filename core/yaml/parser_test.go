package yaml

import (
	"testing"
)

func TestParseSimpleMap(t *testing.T) {
	input := []byte("name: Alice\nage: 30\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
	if len(node.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(node.Pairs))
	}
	if node.Pairs[0].Key != "name" || node.Pairs[0].Value.Value != "Alice" {
		t.Errorf("pair 0: got %q:%q", node.Pairs[0].Key, node.Pairs[0].Value.Value)
	}
	if node.Pairs[1].Key != "age" || node.Pairs[1].Value.Value != "30" {
		t.Errorf("pair 1: got %q:%q", node.Pairs[1].Key, node.Pairs[1].Value.Value)
	}
}

func TestParseNestedMap(t *testing.T) {
	input := []byte("parent:\n  child: value\n  other: data\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
	child := node.Pairs[0].Value
	if child.Type != MappingNode {
		t.Fatalf("expected nested MappingNode, got %s", child.Type)
	}
	if len(child.Pairs) != 2 {
		t.Fatalf("expected 2 nested pairs, got %d", len(child.Pairs))
	}
}

func TestParseSequence(t *testing.T) {
	input := []byte("- one\n- two\n- three\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode {
		t.Fatalf("expected SequenceNode, got %s", node.Type)
	}
	if len(node.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(node.Children))
	}
	if node.Children[0].Value != "one" {
		t.Errorf("child 0: expected 'one', got %q", node.Children[0].Value)
	}
}

func TestParseSequenceOfMaps(t *testing.T) {
	input := []byte("- name: Alice\n  age: 30\n- name: Bob\n  age: 25\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode {
		t.Fatalf("expected SequenceNode, got %s", node.Type)
	}
	if len(node.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(node.Children))
	}
	for i, child := range node.Children {
		if child.Type != MappingNode {
			t.Errorf("child %d: expected MappingNode, got %s", i, child.Type)
		}
	}
}

func TestParseMapWithSequenceValue(t *testing.T) {
	input := []byte("items:\n  - one\n  - two\n  - three\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
	items := node.Pairs[0].Value
	if items.Type != SequenceNode {
		t.Fatalf("expected SequenceNode, got %s", items.Type)
	}
	if len(items.Children) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items.Children))
	}
}

func TestParseFlowSequence(t *testing.T) {
	input := []byte("tags: [api, smoke, fast]\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
	tags := node.Pairs[0].Value
	if tags.Type != SequenceNode {
		t.Fatalf("expected SequenceNode, got %s", tags.Type)
	}
	if len(tags.Children) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags.Children))
	}
}

func TestParseFlowMapping(t *testing.T) {
	input := []byte("config: {a: 1, b: 2}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	config := node.Pairs[0].Value
	if config.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", config.Type)
	}
	if len(config.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(config.Pairs))
	}
}

func TestParseLiteralBlock(t *testing.T) {
	input := []byte("desc: |\n  line1\n  line2\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	desc := node.Pairs[0].Value
	if desc.Type != ScalarNode {
		t.Fatalf("expected ScalarNode, got %s", desc.Type)
	}
	if desc.Value != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got %q", desc.Value)
	}
}

func TestParseFoldedBlock(t *testing.T) {
	input := []byte("desc: >\n  line1\n  line2\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	desc := node.Pairs[0].Value
	if desc.Value != "line1 line2\n" {
		t.Errorf("expected 'line1 line2\\n', got %q", desc.Value)
	}
}

func TestParseEmptyInput(t *testing.T) {
	node, err := Parse([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != ScalarNode {
		t.Errorf("expected ScalarNode for empty input, got %s", node.Type)
	}
}

func TestParseEmptyValue(t *testing.T) {
	input := []byte("key:\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
	val := node.Pairs[0].Value
	if val.Type != ScalarNode || val.Value != "" {
		t.Errorf("expected empty scalar, got %s %q", val.Type, val.Value)
	}
}

func TestParseComments(t *testing.T) {
	input := []byte("# comment\nkey: value # inline\n# end\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode || len(node.Pairs) != 1 {
		t.Fatalf("expected MappingNode with 1 pair, got %s with %d pairs", node.Type, len(node.Pairs))
	}
}

func TestParseFlowSequenceUnterminated(t *testing.T) {
	_, err := Parse([]byte("[a, b"))
	if err == nil {
		t.Error("expected error for unterminated flow sequence")
	}
}

func TestParseFlowMappingUnterminated(t *testing.T) {
	_, err := Parse([]byte("{a: 1"))
	if err == nil {
		t.Error("expected error for unterminated flow mapping")
	}
}

func TestNodeTypeStrings(t *testing.T) {
	tests := []struct {
		t    NodeType
		want string
	}{
		{ScalarNode, "Scalar"},
		{MappingNode, "Mapping"},
		{SequenceNode, "Sequence"},
		{NodeType(99), "Unknown"},
	}
	for _, tt := range tests {
		if tt.t.String() != tt.want {
			t.Errorf("NodeType(%d).String() = %q, want %q", tt.t, tt.t.String(), tt.want)
		}
	}
}

func TestParseTopLevelFlowSequence(t *testing.T) {
	node, err := Parse([]byte("[1, 2, 3]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode || len(node.Children) != 3 {
		t.Errorf("expected sequence with 3 items, got %s with %d", node.Type, len(node.Children))
	}
}

func TestParseTopLevelFlowMapping(t *testing.T) {
	node, err := Parse([]byte("{a: 1, b: 2}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode || len(node.Pairs) != 2 {
		t.Errorf("expected mapping with 2 pairs, got %s with %d", node.Type, len(node.Pairs))
	}
}

func TestParseNestedFlowInBlockSequence(t *testing.T) {
	input := []byte("- {a: 1}\n- [x, y]\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode || len(node.Children) != 2 {
		t.Fatalf("expected sequence with 2 children")
	}
	if node.Children[0].Type != MappingNode {
		t.Errorf("child 0: expected MappingNode, got %s", node.Children[0].Type)
	}
	if node.Children[1].Type != SequenceNode {
		t.Errorf("child 1: expected SequenceNode, got %s", node.Children[1].Type)
	}
}

func TestInterpretScalar(t *testing.T) {
	tests := []struct {
		input string
		want  any
	}{
		{"", nil},
		{"null", nil},
		{"~", nil},
		{"true", true},
		{"false", false},
		{"yes", true},
		{"no", false},
		{"TRUE", true},
		{"FALSE", false},
		{"42", 42},
		{"-7", -7},
		{"+3", 3},
		{"3.14", 3.14},
		{"-0.5", -0.5},
		{"1e2", 100.0},
		{"hello", "hello"},
	}
	for _, tt := range tests {
		got := interpretScalar(tt.input)
		if got != tt.want {
			t.Errorf("interpretScalar(%q) = %v (%T), want %v (%T)", tt.input, got, got, tt.want, tt.want)
		}
	}
}

func TestIsInteger(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"123", true},
		{"-1", true},
		{"+5", true},
		{"12.3", false},
		{"abc", false},
		{"-", false},
		{"+", false},
	}
	for _, tt := range tests {
		if got := isInteger(tt.s); got != tt.want {
			t.Errorf("isInteger(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestIsFloat(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"3.14", true},
		{"1e2", true},
		{"-0.5", true},
		{"1.2.3", false},
		{"+", false},
		{"1E+3", true},
		{"1E-2", true},
		{"abc", false},
		{"1e2e3", false}, // double exponent
	}
	for _, tt := range tests {
		if got := isFloat(tt.s); got != tt.want {
			t.Errorf("isFloat(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestParseFloatExponent(t *testing.T) {
	tests := []struct {
		s    string
		want float64
	}{
		{"1e2", 100.0},
		{"1E-2", 0.01},
		{"-2.5", -2.5},
		{"+1.0", 1.0},
	}
	for _, tt := range tests {
		got := parseFloat(tt.s)
		diff := got - tt.want
		if diff < -0.001 || diff > 0.001 {
			t.Errorf("parseFloat(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestNodeToInterface(t *testing.T) {
	// nil node
	result := nodeToInterface(nil)
	if result != nil {
		t.Errorf("expected nil for nil node, got %v", result)
	}

	// Scalar
	result = nodeToInterface(&Node{Type: ScalarNode, Value: "42"})
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}

	// Mapping
	mapNode := &Node{
		Type: MappingNode,
		Pairs: []KeyValue{
			{Key: "a", Value: &Node{Type: ScalarNode, Value: "1"}},
		},
	}
	result = nodeToInterface(mapNode)
	m, ok := result.(map[string]any)
	if !ok || m["a"] != 1 {
		t.Errorf("expected map with a=1, got %v", result)
	}

	// Sequence
	seqNode := &Node{
		Type: SequenceNode,
		Children: []*Node{
			{Type: ScalarNode, Value: "x"},
		},
	}
	result = nodeToInterface(seqNode)
	s, ok := result.([]any)
	if !ok || len(s) != 1 || s[0] != "x" {
		t.Errorf("expected [x], got %v", result)
	}
}

func TestParseFlowMappingEmptyValue(t *testing.T) {
	input := []byte("{a:}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode || len(node.Pairs) != 1 {
		t.Fatalf("expected mapping with 1 pair, got %s with %d", node.Type, len(node.Pairs))
	}
	if node.Pairs[0].Value.Value != "" {
		t.Errorf("expected empty value, got %q", node.Pairs[0].Value.Value)
	}
}

func TestParseFlowMappingCommaAfterKey(t *testing.T) {
	input := []byte("{a: 1,}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode || len(node.Pairs) != 1 {
		t.Fatalf("expected mapping with 1 pair, got %s with %d", node.Type, len(node.Pairs))
	}
}

func TestParseDeeplyNested(t *testing.T) {
	input := []byte("a:\n  b:\n    c:\n      d: deep\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	a := node.Pairs[0].Value
	if a.Type != MappingNode {
		t.Fatal("a should be mapping")
	}
	b := a.Pairs[0].Value
	if b.Type != MappingNode {
		t.Fatal("b should be mapping")
	}
	c := b.Pairs[0].Value
	if c.Type != MappingNode {
		t.Fatal("c should be mapping")
	}
	d := c.Pairs[0].Value
	if d.Value != "deep" {
		t.Errorf("expected 'deep', got %q", d.Value)
	}
}

func TestParseSequenceWithLiteralBlock(t *testing.T) {
	input := []byte("items:\n  - |\n    block\n    content\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	items := node.Pairs[0].Value
	if items.Type != SequenceNode || len(items.Children) != 1 {
		t.Fatalf("expected sequence with 1 item, got %s %d", items.Type, len(items.Children))
	}
	if items.Children[0].Value != "block\ncontent\n" {
		t.Errorf("expected 'block\\ncontent\\n', got %q", items.Children[0].Value)
	}
}

func TestParseSequenceWithFlowMapping(t *testing.T) {
	input := []byte("- status: 201\n- body.name: {equals: \"Alice\"}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode || len(node.Children) != 2 {
		t.Fatalf("expected sequence with 2 items")
	}
	child1 := node.Children[1]
	if child1.Type != MappingNode {
		t.Fatalf("expected mapping, got %s", child1.Type)
	}
}

func TestParseNestedSequenceInMapping(t *testing.T) {
	input := []byte("a:\n  - x\n  - y\nb: val\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode || len(node.Pairs) != 2 {
		t.Fatalf("expected mapping with 2 pairs, got %d", len(node.Pairs))
	}
	a := node.Pairs[0].Value
	if a.Type != SequenceNode || len(a.Children) != 2 {
		t.Fatalf("expected sequence with 2 items, got %s %d", a.Type, len(a.Children))
	}
}

func TestParseScalarScanError(t *testing.T) {
	// Trigger scanner error
	_, err := Parse([]byte(`"unterminated`))
	if err == nil {
		t.Error("expected error for unterminated quote")
	}
}

func TestParseKeyMethod(t *testing.T) {
	tok := Token{Value: "test_key"}
	if tok.Key() != "test_key" {
		t.Errorf("expected 'test_key', got %q", tok.Key())
	}
}

func TestParseNestedSequence(t *testing.T) {
	input := []byte("items:\n  - - a\n    - b\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	items := node.Pairs[0].Value
	if items.Type != SequenceNode || len(items.Children) != 1 {
		t.Fatalf("expected sequence with 1 child, got %s %d", items.Type, len(items.Children))
	}
}

func TestParseSequenceItemAtEOF(t *testing.T) {
	input := []byte("- ")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode || len(node.Children) != 1 {
		t.Fatalf("expected sequence with 1 child")
	}
	if node.Children[0].Value != "" {
		t.Errorf("expected empty value, got %q", node.Children[0].Value)
	}
}

func TestParseFlowSequenceNested(t *testing.T) {
	input := []byte("[[1, 2], [3, 4]]\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode || len(node.Children) != 2 {
		t.Fatalf("expected sequence with 2 children")
	}
	if node.Children[0].Type != SequenceNode {
		t.Errorf("child 0 should be sequence")
	}
}

func TestParseFlowMappingScalarKey(t *testing.T) {
	// In flow mapping, a bare scalar might appear as key without colon
	input := []byte("{a: [1, 2]}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
	val := node.Pairs[0].Value
	if val.Type != SequenceNode || len(val.Children) != 2 {
		t.Fatalf("expected sequence with 2 children, got %s %d", val.Type, len(val.Children))
	}
}

func TestParseFlowMappingError(t *testing.T) {
	// Flow mapping with unexpected inner token - construct directly
	p := &Parser{
		tokens: []Token{
			{Type: TokenFlowMappingStart, Line: 1, Column: 1},
			{Type: TokenFlowSequenceEnd, Line: 1, Column: 2},
			{Type: TokenEOF, Line: 1, Column: 3},
		},
	}
	_, err := p.parseFlowMapping()
	if err == nil {
		t.Error("expected error for unexpected token in flow mapping")
	}
}

func TestParseFlowMappingNestedValue(t *testing.T) {
	input := []byte("{a: {b: 1}}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	inner := node.Pairs[0].Value
	if inner.Type != MappingNode || len(inner.Pairs) != 1 {
		t.Fatalf("expected nested mapping with 1 pair")
	}
}

func TestParseEmptyMappingValue(t *testing.T) {
	input := []byte("a:\nb: val\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(node.Pairs))
	}
	if node.Pairs[0].Value.Value != "" {
		t.Errorf("expected empty value for 'a', got %q", node.Pairs[0].Value.Value)
	}
}

func TestParseMappingValueAtSameIndent(t *testing.T) {
	// MappingKey at same indent means empty value for previous key
	input := []byte("a:\nb:\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(node.Pairs))
	}
}

func TestParseMappingValueSequenceAtSameIndent(t *testing.T) {
	// SequenceEntry at same indent as key should be empty value
	input := []byte("key:\n- item\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	// This should parse as a mapping with key:"key" value:sequence
	// because the sequence is at the same indent level
	if node.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", node.Type)
	}
}

func TestParseNodeMappingValue(t *testing.T) {
	// MappingValue token type is not produced by our scanner normally,
	// but test the parser's behavior with a constructed token stream
	p := &Parser{
		tokens: []Token{
			{Type: TokenMappingValue, Line: 1, Column: 1},
			{Type: TokenEOF, Line: 1, Column: 2},
		},
	}
	_, err := p.parseDocument()
	if err == nil {
		t.Error("expected error for unexpected MappingValue token")
	}
}

func TestNodeToInterfaceUnknownType(t *testing.T) {
	node := &Node{Type: NodeType(99)}
	result := nodeToInterface(node)
	if result != nil {
		t.Errorf("expected nil for unknown node type, got %v", result)
	}
}

func TestIsFloatDoubleDot(t *testing.T) {
	// Dot in exponent part
	if isFloat("1.2e3.4") {
		t.Error("should not be valid float")
	}
}

func TestParseFloatNoFraction(t *testing.T) {
	f := parseFloat("1e3")
	if f != 1000.0 {
		t.Errorf("expected 1000, got %f", f)
	}
}

func TestParseFlowValueMappingKey(t *testing.T) {
	// Test parseFlowValue receiving a MappingKey token
	p := &Parser{
		tokens: []Token{
			{Type: TokenFlowSequenceStart, Line: 1, Column: 1},
			{Type: TokenMappingKey, Value: "key", Line: 1, Column: 2},
			{Type: TokenFlowSequenceEnd, Line: 1, Column: 5},
			{Type: TokenEOF, Line: 1, Column: 6},
		},
	}
	node, err := p.parseDocument()
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != SequenceNode || len(node.Children) != 1 {
		t.Fatalf("expected sequence with 1 child, got %s %d", node.Type, len(node.Children))
	}
}

func TestParseFlowValueError(t *testing.T) {
	// Test parseFlowValue receiving an unexpected token
	p := &Parser{
		tokens: []Token{
			{Type: TokenFlowSequenceStart, Line: 1, Column: 1},
			{Type: TokenSequenceEntry, Line: 1, Column: 2},
			{Type: TokenEOF, Line: 1, Column: 3},
		},
	}
	_, err := p.parseDocument()
	if err == nil {
		t.Error("expected error for unexpected token in flow value")
	}
}

func TestParseFlowMappingScalarFallback(t *testing.T) {
	// Test flow mapping with scalar token as key
	p := &Parser{
		tokens: []Token{
			{Type: TokenFlowMappingStart, Line: 1, Column: 1},
			{Type: TokenScalar, Value: "bare", Line: 1, Column: 2},
			{Type: TokenFlowMappingEnd, Line: 1, Column: 6},
			{Type: TokenEOF, Line: 1, Column: 7},
		},
	}
	node, err := p.parseDocument()
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != MappingNode || len(node.Pairs) != 1 {
		t.Fatalf("expected mapping with 1 pair, got %s %d", node.Type, len(node.Pairs))
	}
}

func TestParseFlowMappingValueError(t *testing.T) {
	// Flow mapping where value parsing fails
	p := &Parser{
		tokens: []Token{
			{Type: TokenFlowMappingStart, Line: 1, Column: 1},
			{Type: TokenMappingKey, Value: "key", Line: 1, Column: 2},
			{Type: TokenSequenceEntry, Line: 1, Column: 5}, // unexpected in flow value
			{Type: TokenEOF, Line: 1, Column: 6},
		},
	}
	_, err := p.parseDocument()
	if err == nil {
		t.Error("expected error for bad flow value")
	}
}

func TestParseBlockMappingValueError(t *testing.T) {
	// Mapping value that triggers parse error
	p := &Parser{
		tokens: []Token{
			{Type: TokenMappingKey, Value: "key", Line: 1, Column: 1, Indent: 0},
			{Type: TokenFlowSequenceStart, Line: 1, Column: 5},
			// No FlowSequenceEnd -> unterminated
			{Type: TokenEOF, Line: 1, Column: 6},
		},
	}
	_, err := p.parseDocument()
	if err == nil {
		t.Error("expected error for unterminated flow sequence in value")
	}
}

func TestParseBlockSequenceError(t *testing.T) {
	// Sequence item with flow that has error
	p := &Parser{
		tokens: []Token{
			{Type: TokenSequenceEntry, Line: 1, Column: 1, Indent: 0},
			{Type: TokenFlowMappingStart, Line: 1, Column: 3},
			// unterminated
			{Type: TokenEOF, Line: 1, Column: 4},
		},
	}
	_, err := p.parseDocument()
	if err == nil {
		t.Error("expected error for unterminated flow mapping in sequence")
	}
}

func TestParserPeekBeyondEnd(t *testing.T) {
	p := &Parser{tokens: []Token{{Type: TokenEOF}}, pos: 5}
	tok := p.peek()
	if tok.Type != TokenEOF {
		t.Errorf("expected EOF, got %s", tok.Type)
	}
}

func TestParserNextBeyondEnd(t *testing.T) {
	p := &Parser{tokens: []Token{{Type: TokenEOF}}, pos: 5}
	tok := p.next()
	if tok.Type != TokenEOF {
		t.Errorf("expected EOF, got %s", tok.Type)
	}
}

func TestParseNodeEOF(t *testing.T) {
	// Directly test parseNode with EOF
	p := &Parser{
		tokens: []Token{
			{Type: TokenEOF, Line: 1, Column: 1},
		},
	}
	node, err := p.parseNode(-1)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != ScalarNode || node.Value != "" {
		t.Errorf("expected empty scalar for EOF, got %s %q", node.Type, node.Value)
	}
}

func TestParseMappingValueFlowMapping(t *testing.T) {
	// Mapping value that is a flow mapping
	input := []byte("check: {status: 200}\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	val := node.Pairs[0].Value
	if val.Type != MappingNode {
		t.Fatalf("expected MappingNode, got %s", val.Type)
	}
}

func TestParseMappingValueLiteralBlock(t *testing.T) {
	input := []byte("desc: |\n  block\n  content\nnext: val\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(node.Pairs))
	}
	if node.Pairs[0].Value.Value != "block\ncontent\n" {
		t.Errorf("got %q", node.Pairs[0].Value.Value)
	}
}

func TestParseMappingValueFoldedBlock(t *testing.T) {
	input := []byte("desc: >\n  folded\n  text\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Pairs[0].Value.Value != "folded text\n" {
		t.Errorf("got %q", node.Pairs[0].Value.Value)
	}
}

func TestParseMappingValueFlowSequence(t *testing.T) {
	input := []byte("tags: [a, b]\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Pairs[0].Value.Type != SequenceNode {
		t.Fatalf("expected SequenceNode, got %s", node.Pairs[0].Value.Type)
	}
}

func TestParseBlockSequenceNestedSequence(t *testing.T) {
	// Nested sequences at different indent levels
	input := []byte("outer:\n  - inner:\n      - a\n      - b\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	outer := node.Pairs[0].Value
	if outer.Type != SequenceNode {
		t.Fatalf("expected SequenceNode, got %s", outer.Type)
	}
}

func TestParseSequenceWithFoldedBlock(t *testing.T) {
	input := []byte("items:\n  - >\n    folded\n    text\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	items := node.Pairs[0].Value
	if items.Type != SequenceNode || len(items.Children) != 1 {
		t.Fatalf("expected sequence with 1 item")
	}
	if items.Children[0].Value != "folded text\n" {
		t.Errorf("expected 'folded text\\n', got %q", items.Children[0].Value)
	}
}

func TestParseTopLevelScalar(t *testing.T) {
	input := []byte("just a plain scalar\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != ScalarNode || node.Value != "just a plain scalar" {
		t.Errorf("expected scalar 'just a plain scalar', got %s %q", node.Type, node.Value)
	}
}

func TestParseNodeLiteralBlock(t *testing.T) {
	input := []byte("|\n  line1\n  line2\n")
	node, err := Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != ScalarNode {
		t.Fatalf("expected ScalarNode, got %s", node.Type)
	}
	if node.Value != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got %q", node.Value)
	}
}

func TestParseBlockSequenceErrorPropagation(t *testing.T) {
	// Sequence item with error-producing content
	p := &Parser{
		tokens: []Token{
			{Type: TokenSequenceEntry, Line: 1, Column: 1, Indent: 0},
			{Type: TokenMappingKey, Value: "key", Line: 1, Column: 3, Indent: 2},
			{Type: TokenFlowSequenceStart, Line: 1, Column: 7},
			// Missing FlowSequenceEnd -> error
			{Type: TokenEOF, Line: 1, Column: 8},
		},
	}
	_, err := p.parseDocument()
	if err == nil {
		t.Error("expected error")
	}
}

func TestParseExponentialFloat(t *testing.T) {
	// Test float parsing with explicit sign in exponent
	f := parseFloat("1e+3")
	if f != 1000.0 {
		t.Errorf("expected 1000, got %f", f)
	}
}
