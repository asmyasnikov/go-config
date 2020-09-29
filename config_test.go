package config

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type Internal struct {
	Value int `json:"value" description:"struct internal int field"`
}

type TestConfiguration struct {
	IntField1     int      `json:"int_field_1" description:"int field # 1"`
	IntField2     int      `json:"int_field_2" description:"int field # 1"`
	StringField1  string   `json:"string_field_1" description:"string field # 1"`
	StringField2  string   `json:"string_field_2" description:"string field # 2"`
	BoolField1    bool     `json:"bool_field_1" description:"bool field # 1"`
	BoolField2    bool     `json:"bool_field_2" description:"bool field # 2"`
	Float64Field1 float64  `json:"float64_field_1" description:"float64 field # 1"`
	Float64Field2 float64  `json:"float64_field_2" description:"float64 field # 2"`
	Internal      Internal `json:"internal" description:"struct field"`
}

func TestPrintConfig(t *testing.T) {
	configuration := TestConfiguration{
		123,
		-123,
		"some string 1",
		"some string 2",
		true,
		false,
		123.123,
		-123.123,
		Internal{932832},
	}
	expected := `
Test-application (version 1.2.3-test) running with params:

 - IntField1                 : 123
 - IntField2                 : -123
 - StringField1              : some string 1
 - StringField2              : some string 2
 - BoolField1                : true
 - BoolField2                : false
 - Float64Field1             : 123.123000
 - Float64Field2             : -123.123000
 - Internal                 
    - Value                  : 932832

`
	var b bytes.Buffer
	PrintConfig(&b, configuration, "Test-application", "1.2.3-test")
	require.Equal(t, expected, b.String(), "PrintConfig")
	b.Reset()
	PrintConfig(&b, &configuration, "Test-application", "1.2.3-test")
	require.Equal(t, expected, b.String(), "PrintConfig")
}

func TestReadConfig(t *testing.T) {
	os.Setenv("INT_FIELD_2", "-321")
	os.Setenv("STRING_FIELD_1", "test string 1")
	os.Setenv("BOOL_FIELD_2", "true")
	os.Setenv("FLOAT_64_FIELD_1", "123.123")
	os.Setenv("INTERNAL.VALUE", "4444")
	c, err := ReadConfig(
		func() interface{} {
			configuration := &TestConfiguration{
				123,
				-123,
				"some string 1",
				"some string 2",
				true,
				false,
				123.123,
				-123.123,
				Internal{9832},
			}
			return configuration
		},
		"test application",
		"0.0.0-alpha",
		"/tmp/config.json",
	)
	defer os.Remove("/tmp/config.json")
	require.NoError(t, err, "ReadConfig fail")
	configuration, ok := (*c).(*TestConfiguration)
	require.True(t, ok, "Cast fail")
	require.Equal(t, -321, configuration.IntField2, "Set environment variable IntField2 fail")
	require.Equal(t, "test string 1", configuration.StringField1, "Set environment variable StringField1 fail")
	require.True(t, configuration.BoolField2, "Set environment variable BoolField2 fail")
	require.Equal(t, 4444, configuration.Internal.Value, "Set environment variable Internal.Value fail")
}
