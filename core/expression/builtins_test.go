package expression

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

// --- String functions ---

func TestRandomString_Alpha(t *testing.T) {
	val, err := builtinRandomString([]any{"alpha", 5, 10})
	if err != nil {
		t.Fatal(err)
	}
	s := val.(string)
	if len(s) < 5 || len(s) > 10 {
		t.Errorf("length = %d, want [5,10]", len(s))
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			t.Errorf("unexpected character %q in alpha string", r)
		}
	}
}

func TestRandomString_Alphanumeric(t *testing.T) {
	val, err := builtinRandomString([]any{"alphanumeric", 8, 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(val.(string)) != 8 {
		t.Errorf("length = %d, want 8", len(val.(string)))
	}
}

func TestRandomString_InvalidCharset(t *testing.T) {
	_, err := builtinRandomString([]any{"invalid", 1, 5})
	if err == nil {
		t.Fatal("expected error for invalid charset")
	}
}

func TestRandomString_WrongArgCount(t *testing.T) {
	_, err := builtinRandomString([]any{"alpha"})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestRandomString_InvalidRange(t *testing.T) {
	_, err := builtinRandomString([]any{"alpha", 10, 5})
	if err == nil {
		t.Fatal("expected error for min > max")
	}
}

func TestRandomString_NonStringCharset(t *testing.T) {
	_, err := builtinRandomString([]any{123, 1, 5})
	if err == nil {
		t.Fatal("expected error for non-string charset")
	}
}

func TestUUID_Format(t *testing.T) {
	val, err := builtinUUID(nil)
	if err != nil {
		t.Fatal(err)
	}
	s := val.(string)
	// UUID v4: 8-4-4-4-12
	if len(s) != 36 {
		t.Errorf("length = %d, want 36", len(s))
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		t.Errorf("bad format: %s", s)
	}
	// Version nibble should be '4'
	if s[14] != '4' {
		t.Errorf("version nibble = %c, want '4'", s[14])
	}
}

func TestUUID_WrongArgs(t *testing.T) {
	_, err := builtinUUID([]any{"extra"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpper(t *testing.T) {
	val, err := builtinUpper([]any{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "HELLO" {
		t.Errorf("got %v", val)
	}
}

func TestLower(t *testing.T) {
	val, err := builtinLower([]any{"HELLO"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %v", val)
	}
}

func TestTrim(t *testing.T) {
	val, err := builtinTrim([]any{"  hello  "})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %v", val)
	}
}

func TestConcat_Multiple(t *testing.T) {
	val, err := builtinConcat([]any{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "abc" {
		t.Errorf("got %v", val)
	}
}

func TestConcat_Empty(t *testing.T) {
	val, err := builtinConcat(nil)
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("got %v", val)
	}
}

func TestSubstring_Valid(t *testing.T) {
	val, err := builtinSubstring([]any{"hello", 1, 4})
	if err != nil {
		t.Fatal(err)
	}
	if val != "ell" {
		t.Errorf("got %v", val)
	}
}

func TestSubstring_OutOfBounds(t *testing.T) {
	_, err := builtinSubstring([]any{"hi", 0, 10})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubstring_WrongArgs(t *testing.T) {
	_, err := builtinSubstring([]any{"hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReplace(t *testing.T) {
	val, err := builtinReplace([]any{"hello world", "world", "go"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello go" {
		t.Errorf("got %v", val)
	}
}

func TestReplace_WrongArgs(t *testing.T) {
	_, err := builtinReplace([]any{"hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLength(t *testing.T) {
	val, err := builtinLength([]any{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if val != 5 {
		t.Errorf("got %v", val)
	}
}

func TestLength_WrongArgs(t *testing.T) {
	_, err := builtinLength([]any{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpper_WrongArgs(t *testing.T) {
	_, err := builtinUpper(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLower_WrongArgs(t *testing.T) {
	_, err := builtinLower(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTrim_WrongArgs(t *testing.T) {
	_, err := builtinTrim(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Math functions ---

func TestRandomInt_Range(t *testing.T) {
	for i := 0; i < 100; i++ {
		val, err := builtinRandomInt([]any{1, 10})
		if err != nil {
			t.Fatal(err)
		}
		n := val.(int)
		if n < 1 || n > 10 {
			t.Errorf("out of range: %d", n)
		}
	}
}

func TestRandomInt_Equal(t *testing.T) {
	val, err := builtinRandomInt([]any{5, 5})
	if err != nil {
		t.Fatal(err)
	}
	if val != 5 {
		t.Errorf("got %v", val)
	}
}

func TestRandomInt_MinGreaterThanMax(t *testing.T) {
	_, err := builtinRandomInt([]any{10, 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRandomInt_WrongArgs(t *testing.T) {
	_, err := builtinRandomInt([]any{1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRandomFloat_Range(t *testing.T) {
	val, err := builtinRandomFloat([]any{0.0, 1.0})
	if err != nil {
		t.Fatal(err)
	}
	f := val.(float64)
	if f < 0.0 || f > 1.0 {
		t.Errorf("out of range: %f", f)
	}
}

func TestRandomFloat_MinGreaterThanMax(t *testing.T) {
	_, err := builtinRandomFloat([]any{10.0, 1.0})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRandomFloat_WrongArgs(t *testing.T) {
	_, err := builtinRandomFloat([]any{1.0})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAbs(t *testing.T) {
	val, err := builtinAbs([]any{-5.5})
	if err != nil {
		t.Fatal(err)
	}
	if val != 5.5 {
		t.Errorf("got %v", val)
	}
}

func TestAbs_WrongArgs(t *testing.T) {
	_, err := builtinAbs(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCeil(t *testing.T) {
	val, err := builtinCeil([]any{3.2})
	if err != nil {
		t.Fatal(err)
	}
	if val != 4 {
		t.Errorf("got %v", val)
	}
}

func TestCeil_WrongArgs(t *testing.T) {
	_, err := builtinCeil(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFloor(t *testing.T) {
	val, err := builtinFloor([]any{3.9})
	if err != nil {
		t.Fatal(err)
	}
	if val != 3 {
		t.Errorf("got %v", val)
	}
}

func TestFloor_WrongArgs(t *testing.T) {
	_, err := builtinFloor(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRound(t *testing.T) {
	val, err := builtinRound([]any{3.5})
	if err != nil {
		t.Fatal(err)
	}
	if val != 4 {
		t.Errorf("got %v", val)
	}
}

func TestRound_WrongArgs(t *testing.T) {
	_, err := builtinRound(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMin(t *testing.T) {
	val, err := builtinMin([]any{3, 7})
	if err != nil {
		t.Fatal(err)
	}
	if val != 3.0 {
		t.Errorf("got %v", val)
	}
}

func TestMin_WrongArgs(t *testing.T) {
	_, err := builtinMin([]any{1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMax(t *testing.T) {
	val, err := builtinMax([]any{3, 7})
	if err != nil {
		t.Fatal(err)
	}
	if val != 7.0 {
		t.Errorf("got %v", val)
	}
}

func TestMax_WrongArgs(t *testing.T) {
	_, err := builtinMax([]any{1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMod_Valid(t *testing.T) {
	val, err := builtinMod([]any{10, 3})
	if err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Errorf("got %v", val)
	}
}

func TestMod_DivByZero(t *testing.T) {
	_, err := builtinMod([]any{10, 0})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMod_WrongArgs(t *testing.T) {
	_, err := builtinMod([]any{1})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Time functions ---

func TestNow(t *testing.T) {
	val, err := builtinNow(nil)
	if err != nil {
		t.Fatal(err)
	}
	s := val.(string)
	if len(s) < 20 {
		t.Errorf("too short: %s", s)
	}
}

func TestNow_WrongArgs(t *testing.T) {
	_, err := builtinNow([]any{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNowUnix(t *testing.T) {
	val, err := builtinNowUnix(nil)
	if err != nil {
		t.Fatal(err)
	}
	n := val.(int)
	if n < 1000000000 {
		t.Errorf("too small: %d", n)
	}
}

func TestNowUnix_WrongArgs(t *testing.T) {
	_, err := builtinNowUnix([]any{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNowISO(t *testing.T) {
	val, err := builtinNowISO(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(val.(string), "T") {
		t.Errorf("not ISO: %s", val)
	}
}

func TestNowISO_WrongArgs(t *testing.T) {
	_, err := builtinNowISO([]any{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNowRFC3339(t *testing.T) {
	val, err := builtinNowRFC3339(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(val.(string), "T") {
		t.Errorf("not RFC3339: %s", val)
	}
}

func TestNowRFC3339_WrongArgs(t *testing.T) {
	_, err := builtinNowRFC3339([]any{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatTime(t *testing.T) {
	val, err := builtinFormatTime([]any{"2006-01-02"})
	if err != nil {
		t.Fatal(err)
	}
	s := val.(string)
	if len(s) != 10 {
		t.Errorf("format: %s", s)
	}
}

func TestFormatTime_WrongArgs(t *testing.T) {
	_, err := builtinFormatTime(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddDuration(t *testing.T) {
	val, err := builtinAddDuration([]any{"2026-01-01T00:00:00Z", "1h"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(val.(string), "01:00:00") {
		t.Errorf("got %s", val)
	}
}

func TestAddDuration_InvalidTime(t *testing.T) {
	_, err := builtinAddDuration([]any{"not-a-time", "1h"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddDuration_InvalidDuration(t *testing.T) {
	_, err := builtinAddDuration([]any{"2026-01-01T00:00:00Z", "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddDuration_WrongArgs(t *testing.T) {
	_, err := builtinAddDuration([]any{"2026-01-01T00:00:00Z"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Encode functions ---

func TestBase64_RoundTrip(t *testing.T) {
	encoded, err := builtinBase64Encode([]any{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := builtinBase64Decode([]any{encoded})
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "hello world" {
		t.Errorf("got %v", decoded)
	}
}

func TestBase64Encode_WrongArgs(t *testing.T) {
	_, err := builtinBase64Encode(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBase64Decode_Invalid(t *testing.T) {
	_, err := builtinBase64Decode([]any{"not-valid-base64!!!"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBase64Decode_WrongArgs(t *testing.T) {
	_, err := builtinBase64Decode(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestURLEncode(t *testing.T) {
	val, err := builtinURLEncode([]any{"hello world"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello+world" {
		t.Errorf("got %v", val)
	}
}

func TestURLEncode_WrongArgs(t *testing.T) {
	_, err := builtinURLEncode(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestURLDecode(t *testing.T) {
	val, err := builtinURLDecode([]any{"hello+world"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello world" {
		t.Errorf("got %v", val)
	}
}

func TestURLDecode_Invalid(t *testing.T) {
	_, err := builtinURLDecode([]any{"%gg"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestURLDecode_WrongArgs(t *testing.T) {
	_, err := builtinURLDecode(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJSONEncode(t *testing.T) {
	val, err := builtinJSONEncode([]any{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if val != `"hello"` {
		t.Errorf("got %v", val)
	}
}

func TestJSONEncode_WrongArgs(t *testing.T) {
	_, err := builtinJSONEncode(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Crypto functions ---

func TestSHA256_Deterministic(t *testing.T) {
	val1, _ := builtinSHA256([]any{"test"})
	val2, _ := builtinSHA256([]any{"test"})
	if val1 != val2 {
		t.Error("sha256 should be deterministic")
	}
	// Known SHA256 of "test"
	if val1 != "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" {
		t.Errorf("got %v", val1)
	}
}

func TestSHA256_WrongArgs(t *testing.T) {
	_, err := builtinSHA256(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMD5(t *testing.T) {
	val, err := builtinMD5([]any{"test"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "098f6bcd4621d373cade4e832627b4f6" {
		t.Errorf("got %v", val)
	}
}

func TestMD5_WrongArgs(t *testing.T) {
	_, err := builtinMD5(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSHA512(t *testing.T) {
	val, err := builtinSHA512([]any{"test"})
	if err != nil {
		t.Fatal(err)
	}
	s := val.(string)
	if len(s) != 128 { // SHA-512 = 64 bytes = 128 hex chars
		t.Errorf("length = %d", len(s))
	}
}

func TestSHA512_WrongArgs(t *testing.T) {
	_, err := builtinSHA512(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHMACSHA256(t *testing.T) {
	val, err := builtinHMACSHA256([]any{"key", "data"})
	if err != nil {
		t.Fatal(err)
	}
	s := val.(string)
	if len(s) != 64 { // SHA-256 = 32 bytes = 64 hex chars
		t.Errorf("length = %d", len(s))
	}
}

func TestHMACSHA256_WrongArgs(t *testing.T) {
	_, err := builtinHMACSHA256([]any{"key"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Env functions ---

func TestEnv_Present(t *testing.T) {
	t.Setenv("TEST_EXPR_VAR", "hello")
	val, err := builtinEnv([]any{"TEST_EXPR_VAR"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Errorf("got %v", val)
	}
}

func TestEnv_Missing(t *testing.T) {
	_, err := builtinEnv([]any{"NONEXISTENT_EXPR_VAR_12345"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnv_WrongArgs(t *testing.T) {
	_, err := builtinEnv(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvOr_Present(t *testing.T) {
	t.Setenv("TEST_EXPR_OR", "val")
	val, err := builtinEnvOr([]any{"TEST_EXPR_OR", "default"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "val" {
		t.Errorf("got %v", val)
	}
}

func TestEnvOr_Default(t *testing.T) {
	val, err := builtinEnvOr([]any{"NONEXISTENT_EXPR_VAR_12345", "fallback"})
	if err != nil {
		t.Fatal(err)
	}
	if val != "fallback" {
		t.Errorf("got %v", val)
	}
}

func TestEnvOr_WrongArgs(t *testing.T) {
	_, err := builtinEnvOr([]any{"x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Type conversion helpers ---

func TestToInt_String(t *testing.T) {
	_, err := toInt("not a number")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToInt_BadType(t *testing.T) {
	_, err := toInt(true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToFloat_String(t *testing.T) {
	_, err := toFloat("not a number")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToFloat_BadType(t *testing.T) {
	_, err := toFloat(true)
	if err == nil {
		t.Fatal("expected error")
	}
}

// Ensure base64 decode gets valid base64 for the round-trip test
func TestBase64Encode_Output(t *testing.T) {
	val, _ := builtinBase64Encode([]any{"test"})
	s := val.(string)
	_, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Errorf("output is not valid base64: %v", err)
	}
}

// --- Math error paths for non-number types ---

func TestAbs_StringArg(t *testing.T) {
	_, err := builtinAbs([]any{"hello"})
	if err == nil {
		t.Fatal("expected error for string arg")
	}
}

func TestRandomInt_StringArg(t *testing.T) {
	_, err := builtinRandomInt([]any{"a", "b"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRandomFloat_StringArg(t *testing.T) {
	_, err := builtinRandomFloat([]any{"a", "b"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Additional error path tests for builtins ---

func TestRandomString_MinArgNotNumber(t *testing.T) {
	_, err := builtinRandomString([]any{"alpha", "bad", 5})
	if err == nil {
		t.Fatal("expected error for non-number min arg")
	}
}

func TestRandomString_MaxArgNotNumber(t *testing.T) {
	_, err := builtinRandomString([]any{"alpha", 1, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number max arg")
	}
}

func TestRandomInt_MaxArgNotNumber(t *testing.T) {
	_, err := builtinRandomInt([]any{1, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number max arg")
	}
}

func TestRandomFloat_MaxArgNotNumber(t *testing.T) {
	_, err := builtinRandomFloat([]any{1.0, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number max arg")
	}
}

func TestCeil_StringArg(t *testing.T) {
	_, err := builtinCeil([]any{"hello"})
	if err == nil {
		t.Fatal("expected error for string arg")
	}
}

func TestFloor_StringArg(t *testing.T) {
	_, err := builtinFloor([]any{"hello"})
	if err == nil {
		t.Fatal("expected error for string arg")
	}
}

func TestRound_StringArg(t *testing.T) {
	_, err := builtinRound([]any{"hello"})
	if err == nil {
		t.Fatal("expected error for string arg")
	}
}

func TestMin_SecondArgNotNumber(t *testing.T) {
	_, err := builtinMin([]any{1.0, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number second arg")
	}
}

func TestMin_FirstArgNotNumber(t *testing.T) {
	_, err := builtinMin([]any{"bad", 1.0})
	if err == nil {
		t.Fatal("expected error for non-number first arg")
	}
}

func TestMax_SecondArgNotNumber(t *testing.T) {
	_, err := builtinMax([]any{1.0, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number second arg")
	}
}

func TestMax_FirstArgNotNumber(t *testing.T) {
	_, err := builtinMax([]any{"bad", 1.0})
	if err == nil {
		t.Fatal("expected error for non-number first arg")
	}
}

func TestMod_SecondArgNotNumber(t *testing.T) {
	_, err := builtinMod([]any{10, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number second arg")
	}
}

func TestMod_FirstArgNotNumber(t *testing.T) {
	_, err := builtinMod([]any{"bad", 3})
	if err == nil {
		t.Fatal("expected error for non-number first arg")
	}
}

func TestSubstring_StartNotNumber(t *testing.T) {
	_, err := builtinSubstring([]any{"hello", "bad", 3})
	if err == nil {
		t.Fatal("expected error for non-number start arg")
	}
}

func TestSubstring_EndNotNumber(t *testing.T) {
	_, err := builtinSubstring([]any{"hello", 0, "bad"})
	if err == nil {
		t.Fatal("expected error for non-number end arg")
	}
}

func TestToInt_FromFloat(t *testing.T) {
	v, err := toInt(3.7)
	if err != nil {
		t.Fatal(err)
	}
	if v != 3 {
		t.Errorf("got %d, want 3", v)
	}
}

func TestToFloat_FromInt(t *testing.T) {
	v, err := toFloat(42)
	if err != nil {
		t.Fatal(err)
	}
	if v != 42.0 {
		t.Errorf("got %f, want 42.0", v)
	}
}

// --- DB function tests ---

type mockDBOpener struct {
	rows []map[string]any
	err  error
}

func (m *mockDBOpener) Query(dsn, query string) ([]map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

func TestDBQuery_NoDriver(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = nil

	_, err := builtinDBQuery([]any{"dsn", "SELECT 1"})
	if err == nil {
		t.Fatal("expected error for no driver")
	}
	if !strings.Contains(err.Error(), "no database driver") {
		t.Errorf("error = %v", err)
	}
}

func TestDBQuery_WithDriver(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = &mockDBOpener{
		rows: []map[string]any{
			{"id": 1, "name": "alice"},
			{"id": 2, "name": "bob"},
		},
	}

	val, err := builtinDBQuery([]any{"test.db", "SELECT * FROM users"})
	if err != nil {
		t.Fatal(err)
	}
	rows, ok := val.([]map[string]any)
	if !ok {
		t.Fatalf("expected []map[string]any, got %T", val)
	}
	if len(rows) != 2 {
		t.Errorf("rows = %d, want 2", len(rows))
	}
}

func TestDBQuery_WrongArgs(t *testing.T) {
	_, err := builtinDBQuery([]any{"only_one"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDBQuery_DriverError(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = &mockDBOpener{err: fmt.Errorf("connection refused")}

	_, err := builtinDBQuery([]any{"dsn", "SELECT 1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDBQueryOne_Success(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = &mockDBOpener{
		rows: []map[string]any{{"id": 1}},
	}

	val, err := builtinDBQueryOne([]any{"dsn", "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	row, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", val)
	}
	if row["id"] != 1 {
		t.Errorf("id = %v", row["id"])
	}
}

func TestDBQueryOne_NoRows(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = &mockDBOpener{rows: nil}

	_, err := builtinDBQueryOne([]any{"dsn", "SELECT 1"})
	if err == nil {
		t.Fatal("expected error for no rows")
	}
}

func TestDBQueryOne_NoDriver(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = nil

	_, err := builtinDBQueryOne([]any{"dsn", "SELECT 1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDBQueryOne_WrongArgs(t *testing.T) {
	_, err := builtinDBQueryOne([]any{"only_one"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDBCount_Success(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = &mockDBOpener{
		rows: []map[string]any{{"id": 1}, {"id": 2}, {"id": 3}},
	}

	val, err := builtinDBCount([]any{"dsn", "SELECT * FROM users"})
	if err != nil {
		t.Fatal(err)
	}
	if val != 3 {
		t.Errorf("count = %v, want 3", val)
	}
}

func TestDBCount_NoDriver(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()
	dbOpener = nil

	_, err := builtinDBCount([]any{"dsn", "SELECT 1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDBCount_WrongArgs(t *testing.T) {
	_, err := builtinDBCount([]any{"only_one"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterDBOpener(t *testing.T) {
	old := dbOpener
	defer func() { dbOpener = old }()

	mock := &mockDBOpener{}
	RegisterDBOpener(mock)
	if dbOpener != mock {
		t.Fatal("opener not registered")
	}
}
