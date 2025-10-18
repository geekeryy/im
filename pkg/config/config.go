package config

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
)

type Config struct {
	DiscoveryConfig *DiscoveryConfig `env:"IM_DISCOVERY"`
	IMGatewayConfig *IMGatewayConfig `env:"IM_GATEWAY"`
	APIGatewayConfig *APIGatewayConfig `env:"IM_API"`
}

type IMGatewayConfig struct {
	Mode              string      `env:"MODE" default:"dev"`
	Addr              string      `env:"ADDR" default:":8086"`
	RpcAddr           string      `env:"RPC_ADDR" default:"localhost:8087"`
	RedisConfig       RedisConfig `env:"REDIS"`
	DiscoveryEndpoint string      `env:"DISCOVERY_ENDPOINT" default:"localhost:8085"`
}

type DiscoveryConfig struct {
	Mode        string      `env:"MODE" default:"dev"`
	Addr        string      `env:"ADDR" default:":8085"`
	RedisConfig RedisConfig `env:"REDIS"`
}

type APIGatewayConfig struct {
	Mode        string      `env:"MODE" default:"dev"`
	Addr        string      `env:"ADDR" default:":8088"`
	RedisConfig RedisConfig `env:"REDIS"`
	MysqlConfig MysqlConfig `env:"MYSQL"`
}
type MysqlConfig struct {
	Addr     string `env:"ADDR" default:"127.0.0.1:3306"`
	Username string `env:"USERNAME" default:"root"`
	Password string `env:"PASSWORD" default:"root"`
	DB       string `env:"DB" default:"im"`
}

type RedisConfig struct {
	Addr     string `env:"ADDR" default:"127.0.0.1:6379"`
	Password string `env:"PASSWORD" default:"root"`
	DB       int    `env:"DB" default:"0"`
}

func NewConf() *Config {
	conf := Config{}
	Unmarshal(&conf)
	return &conf
}

func (conf *Config) GetDiscoveryConfig() *DiscoveryConfig {
	return conf.DiscoveryConfig
}
func (conf *Config) GetIMGatewayConfig() *IMGatewayConfig {
	return conf.IMGatewayConfig
}
func (conf *Config) GetAPIGatewayConfig() *APIGatewayConfig {
	return conf.APIGatewayConfig
}

func Unmarshal(conf any) {
	v := reflect.ValueOf(conf)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	unmarshal(v, "")
}

func unmarshal(v reflect.Value, envPrefix string) {
	if v.Kind() == reflect.Ptr {
		unmarshal(v.Elem(), envPrefix)
		return
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)

		envName := fieldType.Tag.Get("env")
		if len(envPrefix) > 0 {
			envName = fmt.Sprintf("%s_%s", envPrefix, envName)
		}
		envValue := os.Getenv(envName)
		if len(envValue) == 0 {
			envValue = fieldType.Tag.Get("default")
		}
		switch fieldValue.Kind() {
		case reflect.Struct:
			unmarshal(fieldValue, envName)
		case reflect.Ptr:
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldType.Type.Elem()))
			}
			unmarshal(fieldValue.Elem(), envName)
		case reflect.String:
			fieldValue.SetString(envValue)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			s, err := strconv.ParseInt(envValue, 10, 64)
			if err != nil {
				log.Printf("failed to convert env value to int: %v", err)
			}
			fieldValue.SetInt(s)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			s, err := strconv.ParseUint(envValue, 10, 64)
			if err != nil {
				log.Printf("failed to convert env value to uint: %v", err)
			}
			fieldValue.SetUint(s)
		case reflect.Float32, reflect.Float64:
			s, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				log.Printf("failed to convert env value to float: %v", err)
			}
			fieldValue.SetFloat(s)
		case reflect.Bool:
			b, err := strconv.ParseBool(envValue)
			if err != nil {
				log.Printf("failed to convert env value to bool: %v", err)
			}
			fieldValue.SetBool(b)
		default:
			log.Printf("unsupported type: %v", fieldValue.Kind())
		}
	}

}
