package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/rs/zerolog/log"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
)

var (
	flagValues = make(map[string]interface{})
)

func printConfig(w io.Writer, prefix string, i interface{}) {
	format := prefix + " - %-" + strconv.Itoa(25-len(prefix)) + "s"
	values := reflect.ValueOf(i)
	if values.Kind() == reflect.Ptr {
		values = reflect.Indirect(values)
	}
	if values.Kind() == reflect.Interface {
		values = values.Elem().Elem()
	}
	if values.Kind() != reflect.Struct {
		log.Fatal().Caller().Msgf("unexpected type: %s (%+v)", values.Kind(), i)
	}
	typeOfConfig := values.Type()
	for i := 0; i < values.NumField(); i++ {
		switch typeOfConfig.Field(i).Type.Kind() {
		case reflect.String:
			{
				fmt.Fprintf(w, format + " : %s\n", typeOfConfig.Field(i).Name, values.Field(i).Interface())
			}
		case reflect.Int:
			{
				fmt.Fprintf(w, format + " : %d\n", typeOfConfig.Field(i).Name, values.Field(i).Interface())
			}
		case reflect.Float64:
			{
				fmt.Fprintf(w, format + " : %f\n", typeOfConfig.Field(i).Name, values.Field(i).Interface())
			}
		case reflect.Bool:
			{
				fmt.Fprintf(w, format + " : %t\n", typeOfConfig.Field(i).Name, values.Field(i).Interface())
			}
		case reflect.Struct:
			{
				fmt.Fprintf(w, format + "\n", typeOfConfig.Field(i).Name)
				printConfig(w, prefix+"   ", values.Field(i).Interface())
			}
		}
	}
}

// PrintConfig print current config
func PrintConfig(w io.Writer, config interface{}, applicationName, packageVersionString string) {
	fmt.Fprintf(w, "\n%s (version %s) running with params:\n\n", applicationName, packageVersionString)
	printConfig(w, "", config)
	fmt.Fprintln(w)
}

// Reads config from environment variables, which names are equals to config fields' names
func readConfigFromFlag(prefix string, config interface{}) {
	values := reflect.ValueOf(config)
	if values.Kind() == reflect.Ptr {
		values = reflect.Indirect(values)
	}
	typeOfConfig := values.Type()
	for i := 0; i < values.NumField(); i++ {
		name := prefix + strcase.ToKebab(typeOfConfig.Field(i).Name)
		v, ok := flagValues[name]
		if ok {
			switch typeOfConfig.Field(i).Type.Kind() {
			case reflect.String:
				value := v.(*string)
				values.Field(i).SetString(*value)
			case reflect.Int:
				value := v.(*int)
				values.Field(i).SetInt(int64(*value))
			case reflect.Float64:
				value := v.(*float64)
				values.Field(i).SetFloat(*value)
			case reflect.Bool:
				value := v.(*bool)
				values.Field(i).SetBool(*value)
			case reflect.Struct:
				readConfigFromFlag(name + "-", values.Field(i).Addr().Interface())
			}
		}
	}
}

func initFlagsByConfig(prefix string, config interface{}) {
	values := reflect.ValueOf(config)
	if values.Kind() == reflect.Ptr {
		values = reflect.Indirect(values)
	}
	typeOfConfig := values.Type()
	for i := 0; i < values.NumField(); i++ {
		name := prefix + strcase.ToKebab(typeOfConfig.Field(i).Name)
		tag, ok := typeOfConfig.Field(i).Tag.Lookup("description")
		if !ok {
			log.Error().Caller().Str("field", typeOfConfig.Field(i).Name).Msg("No description")
			continue
		}
		if len(tag) > 0 {
			switch typeOfConfig.Field(i).Type.Kind() {
			case reflect.String:
				flagValues[name] = flag.String(name, values.Field(i).String(), tag)
			case reflect.Int:
				flagValues[name] = flag.Int(name, int(values.Field(i).Int()), tag)
			case reflect.Float64:
				flagValues[name] = flag.Float64(name, values.Field(i).Float(), tag)
			case reflect.Bool:
				flagValues[name] = flag.Bool(name, values.Field(i).Bool(), tag)
			case reflect.Struct:
				initFlagsByConfig(name+"-", values.Field(i).Addr().Interface())
			}
		}
	}
}

// ReadConfig read config by default path or by flag -config
func ReadConfigWithSaver(createDefault func() interface{}, applicationName, packageVersionString, defaultConfigPath string) (*interface{}, func() error, error) {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s (version %s)\n\n", applicationName, packageVersionString)
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	var config = createDefault()

	path := flag.String("config", defaultConfigPath, "path to config")

	initFlagsByConfig("", config)

	flag.Parse()

	jsonConfig, err := ioutil.ReadFile(*path)
	if err != nil {
		log.Error().Caller().Err(err).Msg("")
	} else {
		if err := json.Unmarshal(jsonConfig, &config); err != nil {
			log.Error().Caller().Err(err).Msg("")
		}
	}

	save := func() error {
		bytes, err := json.MarshalIndent(config, "", "\t")
		if err != nil {
			log.Fatal().Caller().Err(err).Msg("")
			return err
		}

		if err := ioutil.WriteFile(*path, bytes, 0666); err != nil {
			log.Error().Caller().Err(err).Str("path", *path).Msg("Save config")
			return err
		}
		return nil
	}

	defer func() {
		if err := save(); err != nil {
			log.Error().Caller().Err(err).Str("path", *path).Msg("Save config")
		}
	}()

	readConfigFromFlag("", config)

	readConfigFromEnv("", config)

	return &config, save, nil
}

// ReadConfig read config by default path or by flag -config
func ReadConfig(createDefault func() interface{}, applicationName, packageVersionString, defaultConfigPath string) (*interface{}, error) {
	config, _, err := ReadConfigWithSaver(createDefault, applicationName, packageVersionString, defaultConfigPath)
	return config, err
}

// Reads config from environment variables, which names are equals to config fields' names
func readConfigFromEnv(prefix string, config interface{}) {
	values := reflect.ValueOf(config)
	if values.Kind() == reflect.Ptr {
		values = reflect.Indirect(values)
	}
	typeOfConfig := values.Type()
	for i := 0; i < values.NumField(); i++ {
		name := prefix + strcase.ToScreamingSnake(typeOfConfig.Field(i).Name)
		v := os.Getenv(name)
		switch typeOfConfig.Field(i).Type.Kind() {
		case reflect.String:
			if v != "" {
				reflect.Indirect(values.Field(i)).SetString(v)
			}
		case reflect.Int:
			if v != "" {
				val, err := strconv.Atoi(v)
				if err != nil {
					fmt.Printf(`Env: wrong value type for field "%s", need int\n`, name)
					continue
				}
				values.Field(i).SetInt(int64(val))
			}
		case reflect.Float64:
			if v != "" {
				val, err := strconv.ParseFloat(v, 64)
				if err != nil {
					fmt.Printf(`Env: wrong value type for field "%s", need int\n`, name)
					continue
				}
				values.Field(i).SetFloat(val)
			}
		case reflect.Bool:
			if v != "" {
				val, err := strconv.ParseBool(v)
				if err != nil {
					fmt.Printf(`Env: wrong value type for field "%s", need bool\n`, name)
					continue
				}
				values.Field(i).SetBool(val)
			}
		case reflect.Struct:
			readConfigFromEnv(name + ".", values.Field(i).Addr().Interface())
		}
	}
}
