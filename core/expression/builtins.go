package expression

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand/v2"
	"net/url"
	"os"
	"strings"
	"time"
)

// registerBuiltins registers all built-in functions in the given registry.
func registerBuiltins(r *Registry) {
	// String functions
	_ = r.Register("random_string", builtinRandomString)
	_ = r.Register("uuid", builtinUUID)
	_ = r.Register("upper", builtinUpper)
	_ = r.Register("lower", builtinLower)
	_ = r.Register("trim", builtinTrim)
	_ = r.Register("concat", builtinConcat)
	_ = r.Register("substring", builtinSubstring)
	_ = r.Register("replace", builtinReplace)
	_ = r.Register("length", builtinLength)

	// Math functions
	_ = r.Register("random_int", builtinRandomInt)
	_ = r.Register("random_float", builtinRandomFloat)
	_ = r.Register("abs", builtinAbs)
	_ = r.Register("ceil", builtinCeil)
	_ = r.Register("floor", builtinFloor)
	_ = r.Register("round", builtinRound)
	_ = r.Register("min", builtinMin)
	_ = r.Register("max", builtinMax)
	_ = r.Register("mod", builtinMod)

	// Time functions
	_ = r.Register("now", builtinNow)
	_ = r.Register("now_unix", builtinNowUnix)
	_ = r.Register("now_iso", builtinNowISO)
	_ = r.Register("now_rfc3339", builtinNowRFC3339)
	_ = r.Register("format_time", builtinFormatTime)
	_ = r.Register("add_duration", builtinAddDuration)

	// Encode functions
	_ = r.Register("base64_encode", builtinBase64Encode)
	_ = r.Register("base64_decode", builtinBase64Decode)
	_ = r.Register("url_encode", builtinURLEncode)
	_ = r.Register("url_decode", builtinURLDecode)
	_ = r.Register("json_encode", builtinJSONEncode)

	// Crypto functions
	_ = r.Register("md5", builtinMD5)
	_ = r.Register("sha256", builtinSHA256)
	_ = r.Register("sha512", builtinSHA512)
	_ = r.Register("hmac_sha256", builtinHMACSHA256)

	// Env functions
	_ = r.Register("env", builtinEnv)
	_ = r.Register("env_or", builtinEnvOr)

	// DB functions — require a driver to be registered via RegisterDBOpener.
	_ = r.Register("db_query", builtinDBQuery)
	_ = r.Register("db_query_one", builtinDBQueryOne)
	_ = r.Register("db_count", builtinDBCount)
}

// --- String functions ---

var charsets = map[string]string{
	"alpha":        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"alphanumeric": "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	"numeric":      "0123456789",
	"hex":          "0123456789abcdef",
	"ascii":        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?/~`",
}

func builtinRandomString(args []any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("random_string: expected 3 args (charset, min, max), got %d", len(args))
	}
	csName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("random_string: charset must be a string")
	}
	cs, ok := charsets[csName]
	if !ok {
		return nil, fmt.Errorf("random_string: unknown charset %q (valid: alpha, alphanumeric, numeric, hex, ascii)", csName)
	}
	minLen, err := toInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("random_string: min: %w", err)
	}
	maxLen, err := toInt(args[2])
	if err != nil {
		return nil, fmt.Errorf("random_string: max: %w", err)
	}
	if minLen < 0 || maxLen < minLen {
		return nil, fmt.Errorf("random_string: invalid range [%d, %d]", minLen, maxLen)
	}

	length := minLen
	if maxLen > minLen {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(maxLen-minLen+1)))
		length = minLen + int(n.Int64())
	}

	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(cs))))
		result[i] = cs[n.Int64()]
	}
	return string(result), nil
}

func builtinUUID(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("uuid: expected 0 args, got %d", len(args))
	}
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return nil, fmt.Errorf("uuid: %w", err)
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}

func builtinUpper(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("upper: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("upper: %w", err)
	}
	return strings.ToUpper(s), nil
}

func builtinLower(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("lower: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("lower: %w", err)
	}
	return strings.ToLower(s), nil
}

func builtinTrim(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("trim: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("trim: %w", err)
	}
	return strings.TrimSpace(s), nil
}

func builtinConcat(args []any) (any, error) {
	var b strings.Builder
	for _, a := range args {
		b.WriteString(fmt.Sprintf("%v", a))
	}
	return b.String(), nil
}

func builtinSubstring(args []any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("substring: expected 3 args (string, start, end), got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("substring: %w", err)
	}
	start, err := toInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("substring: start: %w", err)
	}
	end, err := toInt(args[2])
	if err != nil {
		return nil, fmt.Errorf("substring: end: %w", err)
	}
	runes := []rune(s)
	if start < 0 || end > len(runes) || start > end {
		return nil, fmt.Errorf("substring: index out of bounds [%d:%d] for string of length %d", start, end, len(runes))
	}
	return string(runes[start:end]), nil
}

func builtinReplace(args []any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("replace: expected 3 args (string, old, new), got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("replace: %w", err)
	}
	old, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("replace: old: %w", err)
	}
	newStr, err := toString(args[2])
	if err != nil {
		return nil, fmt.Errorf("replace: new: %w", err)
	}
	return strings.ReplaceAll(s, old, newStr), nil
}

func builtinLength(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("length: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("length: %w", err)
	}
	return len([]rune(s)), nil
}

// --- Math functions ---

func builtinRandomInt(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("random_int: expected 2 args (min, max), got %d", len(args))
	}
	minVal, err := toInt(args[0])
	if err != nil {
		return nil, fmt.Errorf("random_int: min: %w", err)
	}
	maxVal, err := toInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("random_int: max: %w", err)
	}
	if minVal > maxVal {
		return nil, fmt.Errorf("random_int: min (%d) > max (%d)", minVal, maxVal)
	}
	if minVal == maxVal {
		return minVal, nil
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(maxVal-minVal+1)))
	return minVal + int(n.Int64()), nil
}

func builtinRandomFloat(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("random_float: expected 2 args (min, max), got %d", len(args))
	}
	minVal, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("random_float: min: %w", err)
	}
	maxVal, err := toFloat(args[1])
	if err != nil {
		return nil, fmt.Errorf("random_float: max: %w", err)
	}
	if minVal > maxVal {
		return nil, fmt.Errorf("random_float: min (%f) > max (%f)", minVal, maxVal)
	}
	return minVal + mrand.Float64()*(maxVal-minVal), nil
}

func builtinAbs(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("abs: expected 1 arg, got %d", len(args))
	}
	v, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}
	return math.Abs(v), nil
}

func builtinCeil(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ceil: expected 1 arg, got %d", len(args))
	}
	v, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("ceil: %w", err)
	}
	return int(math.Ceil(v)), nil
}

func builtinFloor(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("floor: expected 1 arg, got %d", len(args))
	}
	v, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("floor: %w", err)
	}
	return int(math.Floor(v)), nil
}

func builtinRound(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("round: expected 1 arg, got %d", len(args))
	}
	v, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("round: %w", err)
	}
	return int(math.Round(v)), nil
}

func builtinMin(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("min: expected 2 args, got %d", len(args))
	}
	a, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("min: %w", err)
	}
	b, err := toFloat(args[1])
	if err != nil {
		return nil, fmt.Errorf("min: %w", err)
	}
	return math.Min(a, b), nil
}

func builtinMax(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("max: expected 2 args, got %d", len(args))
	}
	a, err := toFloat(args[0])
	if err != nil {
		return nil, fmt.Errorf("max: %w", err)
	}
	b, err := toFloat(args[1])
	if err != nil {
		return nil, fmt.Errorf("max: %w", err)
	}
	return math.Max(a, b), nil
}

func builtinMod(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("mod: expected 2 args, got %d", len(args))
	}
	a, err := toInt(args[0])
	if err != nil {
		return nil, fmt.Errorf("mod: %w", err)
	}
	b, err := toInt(args[1])
	if err != nil {
		return nil, fmt.Errorf("mod: %w", err)
	}
	if b == 0 {
		return nil, fmt.Errorf("mod: division by zero")
	}
	return a % b, nil
}

// --- Time functions ---

func builtinNow(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("now: expected 0 args, got %d", len(args))
	}
	return time.Now().Format(time.RFC3339), nil
}

func builtinNowUnix(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("now_unix: expected 0 args, got %d", len(args))
	}
	return int(time.Now().Unix()), nil
}

func builtinNowISO(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("now_iso: expected 0 args, got %d", len(args))
	}
	return time.Now().Format("2006-01-02T15:04:05Z07:00"), nil
}

func builtinNowRFC3339(args []any) (any, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("now_rfc3339: expected 0 args, got %d", len(args))
	}
	return time.Now().Format(time.RFC3339Nano), nil
}

func builtinFormatTime(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("format_time: expected 1 arg (layout), got %d", len(args))
	}
	layout, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("format_time: %w", err)
	}
	return time.Now().Format(layout), nil
}

func builtinAddDuration(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("add_duration: expected 2 args (time_str, duration), got %d", len(args))
	}
	timeStr, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("add_duration: time: %w", err)
	}
	durStr, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("add_duration: duration: %w", err)
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, fmt.Errorf("add_duration: invalid time %q: %w", timeStr, err)
	}
	d, err := time.ParseDuration(durStr)
	if err != nil {
		return nil, fmt.Errorf("add_duration: invalid duration %q: %w", durStr, err)
	}
	return t.Add(d).Format(time.RFC3339), nil
}

// --- Encode functions ---

func builtinBase64Encode(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("base64_encode: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("base64_encode: %w", err)
	}
	return base64.StdEncoding.EncodeToString([]byte(s)), nil
}

func builtinBase64Decode(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("base64_decode: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("base64_decode: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64_decode: %w", err)
	}
	return string(decoded), nil
}

func builtinURLEncode(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("url_encode: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("url_encode: %w", err)
	}
	return url.QueryEscape(s), nil
}

func builtinURLDecode(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("url_decode: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("url_decode: %w", err)
	}
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return nil, fmt.Errorf("url_decode: %w", err)
	}
	return decoded, nil
}

func builtinJSONEncode(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("json_encode: expected 1 arg, got %d", len(args))
	}
	data, err := json.Marshal(args[0])
	if err != nil {
		return nil, fmt.Errorf("json_encode: %w", err)
	}
	return string(data), nil
}

// --- Crypto functions ---

func builtinMD5(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("md5: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("md5: %w", err)
	}
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:]), nil
}

func builtinSHA256(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sha256: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("sha256: %w", err)
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:]), nil
}

func builtinSHA512(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sha512: expected 1 arg, got %d", len(args))
	}
	s, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("sha512: %w", err)
	}
	h := sha512.Sum512([]byte(s))
	return hex.EncodeToString(h[:]), nil
}

func builtinHMACSHA256(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("hmac_sha256: expected 2 args (key, data), got %d", len(args))
	}
	key, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("hmac_sha256: key: %w", err)
	}
	data, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("hmac_sha256: data: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// --- Env functions ---

func builtinEnv(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("env: expected 1 arg, got %d", len(args))
	}
	name, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}
	val, ok := os.LookupEnv(name)
	if !ok {
		return nil, fmt.Errorf("env: variable %q not set", name)
	}
	return val, nil
}

func builtinEnvOr(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("env_or: expected 2 args (name, default), got %d", len(args))
	}
	name, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("env_or: %w", err)
	}
	def, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("env_or: default: %w", err)
	}
	val, ok := os.LookupEnv(name)
	if !ok {
		return def, nil
	}
	return val, nil
}

// --- Type conversion helpers ---

func toInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case float64:
		return int(val), nil
	case string:
		// Not supported — callers should pass numbers.
		return 0, fmt.Errorf("expected number, got string %q", val)
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

func toFloat(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case string:
		return 0, fmt.Errorf("expected number, got string %q", val)
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

func toString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

// --- DB functions ---

// DBOpener opens a database connection and executes a query.
// This is the interface that database drivers must implement to enable
// db_query, db_query_one, and db_count expression functions.
type DBOpener interface {
	// Query executes a query and returns all rows as a slice of maps.
	Query(dsn, query string) ([]map[string]any, error)
}

// dbOpener is the global DB opener. Nil until a driver is registered.
var dbOpener DBOpener

// RegisterDBOpener registers a database opener for use by db_* functions.
// This should be called at application startup by a database driver package.
func RegisterDBOpener(opener DBOpener) {
	dbOpener = opener
}

func builtinDBQuery(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("db_query: expected 2 args (dsn, query), got %d", len(args))
	}
	if dbOpener == nil {
		return nil, fmt.Errorf("db_query: no database driver registered (call expression.RegisterDBOpener)")
	}
	dsn, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("db_query: dsn: %w", err)
	}
	query, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("db_query: query: %w", err)
	}
	rows, err := dbOpener.Query(dsn, query)
	if err != nil {
		return nil, fmt.Errorf("db_query: %w", err)
	}
	return rows, nil
}

func builtinDBQueryOne(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("db_query_one: expected 2 args (dsn, query), got %d", len(args))
	}
	if dbOpener == nil {
		return nil, fmt.Errorf("db_query_one: no database driver registered (call expression.RegisterDBOpener)")
	}
	dsn, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("db_query_one: dsn: %w", err)
	}
	query, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("db_query_one: query: %w", err)
	}
	rows, err := dbOpener.Query(dsn, query)
	if err != nil {
		return nil, fmt.Errorf("db_query_one: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("db_query_one: no rows returned")
	}
	return rows[0], nil
}

func builtinDBCount(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("db_count: expected 2 args (dsn, query), got %d", len(args))
	}
	if dbOpener == nil {
		return nil, fmt.Errorf("db_count: no database driver registered (call expression.RegisterDBOpener)")
	}
	dsn, err := toString(args[0])
	if err != nil {
		return nil, fmt.Errorf("db_count: dsn: %w", err)
	}
	query, err := toString(args[1])
	if err != nil {
		return nil, fmt.Errorf("db_count: query: %w", err)
	}
	rows, err := dbOpener.Query(dsn, query)
	if err != nil {
		return nil, fmt.Errorf("db_count: %w", err)
	}
	return len(rows), nil
}
