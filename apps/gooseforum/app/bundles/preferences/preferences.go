package preferences

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/fileopt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

// Viper 库实例
var current atomic.Pointer[viper.Viper]
var configWriteMu sync.Mutex
var revision atomic.Uint64
var validators []func(map[string]any) error

func Revision() uint64 { return revision.Load() }
func AddValidator(v func(map[string]any) error) {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	validators = append(validators, v)
}
func validate(next *viper.Viper) error {
	for _, v := range validators {
		if err := v(next.AllSettings()); err != nil {
			return err
		}
	}
	return nil
}

var configPath string

//go:embed config.templ.toml
var configTempl []byte

func GenerateConfig() ([]byte, error) {
	signingKey := algorithm.SafeGenerateSigningKey(32)

	var b bytes.Buffer
	t := template.New("config.templ.toml")
	t = template.Must(t.Parse(string(configTempl)))
	err := t.Execute(&b, map[string]any{
		"SigningKey": signingKey,
	})
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// 初始化配置信息，完成对环境变量以及 conf 信息的加载
func init() {
	cfgPath := "config.toml"
	wd, _ := os.Getwd()
	if IsTestMode() {
		dir, err := findConfigDirTest(wd, 6)
		if err != nil {
			slog.Error("preferences.test.search", "err", err)
			dir = wd
		}
		cfgPath = filepath.Join(dir, "config.toml")
		if !fileopt.IsExist(cfgPath) {
			configData, err := GenerateConfig()
			if err != nil {
				slog.Error("preferences.test.generate", "err", err)
			} else if e := fileopt.PutContents(cfgPath, configData); e != nil {
				slog.Error("preferences.test.init", "err", e)
			} else {
				slog.Info("preferences.test.init", "path", cfgPath)
			}
		}
	} else if !fileopt.IsExist(cfgPath) {
		configData, err := GenerateConfig()
		if err != nil {
			slog.Error("preferences.generate", "err", err)
		} else if err := fileopt.PutContents(cfgPath, configData); err != nil {
			slog.Error("preferences.init", "err", err)
		}
	}
	v := viper.New()
	v.SetConfigType("toml")
	v.AddConfigPath(filepath.Dir(cfgPath))
	configFlag := flag.String("config", cfgPath, "path to config file")
	v.SetConfigFile(*configFlag)
	if err := v.ReadInConfig(); err != nil {
		slog.Warn("ReadInConfig", "err", err)
	}
	configPath = *configFlag
	current.Store(v)
	revision.Add(1)
}

func internalGet(path string, defaultValue ...any) any {
	v := current.Load()
	// conf 或者环境变量不存在的情况
	if !v.IsSet(path) || v.Get(path) == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return nil
	}
	return cloneValue(v.Get(path))
}

func IsSet(path string) bool {
	v := current.Load()
	return v.IsSet(path) && v.Get(path) != nil
}

// GetRaw returns the raw setting value without type coercion. It is used to
// read structured settings such as the OIDC client list.
func GetRaw(path string) any {
	return internalGet(path)
}

func Set(path string, value any) {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	next := viper.New()
	_ = next.MergeConfigMap(All())
	next.Set(path, cloneValue(value))
	if err := validate(next); err != nil {
		slog.Warn("configuration rejected", "err", err)
		return
	}
	current.Store(next)
	revision.Add(1)
}

var watchConfigOnce sync.Once

// OpenConfigChangeEvent 开启监控配置文件⌚️
func OpenConfigChangeEvent() {
	watchConfigOnce.Do(func() {
		// The watcher is never published: Viper mutates it while reading the file.
		watcher := viper.New()
		watcher.SetConfigFile(configPath)
		watcher.SetConfigType("toml")
		_ = watcher.ReadInConfig()
		watcher.OnConfigChange(func(e fsnotify.Event) {
			configWriteMu.Lock()
			next := viper.New()
			next.SetConfigFile(configPath)
			next.SetConfigType("toml")
			err := next.ReadInConfig()
			if err == nil {
				err = validate(next)
			}
			if err == nil {
				current.Store(next)
				revision.Add(1)
			}
			configWriteMu.Unlock()
			if err != nil {
				slog.Warn("configuration reload rejected", "err", err)
				return
			}
			runEvent(e)
		})
		watcher.WatchConfig()
	})
}

var eventManagerLock sync.Mutex

var eventList []func(e fsnotify.Event)

func AddWatch(event func(e fsnotify.Event)) {
	eventManagerLock.Lock()
	defer eventManagerLock.Unlock()
	eventList = append(eventList, event)
}

func runEvent(e fsnotify.Event) {
	eventManagerLock.Lock()
	defer func() {
		eventManagerLock.Unlock()
		if r := recover(); r != nil {
			slog.Error("recover", "r", r)
		}
	}()
	for _, item := range eventList {
		item(e)
	}
}

// Get returns a string setting and supports dot-separated paths.
func Get(path string, defaultValue ...any) string {
	return GetString(path, defaultValue...)
}

// GetString returns a string setting.
func GetString(path string, defaultValue ...any) string {
	return cast.ToString(internalGet(path, defaultValue...))
}

// GetInt returns an int setting.
func GetInt(path string, defaultValue ...any) int {
	return cast.ToInt(internalGet(path, defaultValue...))
}

// GetFloat64 returns a float64 setting.
func GetFloat64(path string, defaultValue ...any) float64 {
	return cast.ToFloat64(internalGet(path, defaultValue...))
}

// GetInt64 returns an int64 setting.
func GetInt64(path string, defaultValue ...any) int64 {
	return cast.ToInt64(internalGet(path, defaultValue...))
}

// GetUint returns a uint setting.
func GetUint(path string, defaultValue ...any) uint {
	return cast.ToUint(internalGet(path, defaultValue...))
}

// GetBool returns a bool setting.
func GetBool(path string, defaultValue ...any) bool {
	return cast.ToBool(internalGet(path, defaultValue...))
}

// GetStringMapString returns a string map setting.
func GetStringMapString(path string) map[string]string {
	return cast.ToStringMapString(internalGet(path))
}

// GetStringSlice returns a string slice setting.
func GetStringSlice(path string) []string {
	return cast.ToStringSlice(internalGet(path))
}

// GetIntSlice returns an int slice setting.
func GetIntSlice(path string) []int {
	return cast.ToIntSlice(internalGet(path))
}

// All returns all loaded settings.
func All() map[string]any {
	return cloneValue(current.Load().AllSettings()).(map[string]any)
}

func IsTestMode() bool {
	if strings.HasSuffix(os.Args[0], ".test") {
		return true
	}
	for _, a := range os.Args {
		if strings.HasPrefix(a, "-test.") {
			return true
		}
	}
	return false
}

func findConfigDirTest(start string, maxDepth int) (string, error) {
	if fileopt.IsExist(filepath.Join(start, "config.toml")) {
		return start, nil
	}
	if fileopt.IsExist(filepath.Join(start, "go.mod")) {
		return start, nil
	}
	cur := start
	for range maxDepth {
		next := filepath.Dir(cur)
		if next == cur {
			break
		}
		cur = next
		if fileopt.IsExist(filepath.Join(cur, "go.mod")) {
			return cur, nil
		}
	}
	return "", fmt.Errorf("preferences: test mode cannot find go.mod within %d levels from %s", maxDepth, start)
}

// Copy composite settings at the public boundary; callers cannot mutate a published snapshot.
func cloneValue(value any) any {
	if value == nil {
		return nil
	}
	return cloneReflect(reflect.ValueOf(value)).Interface()
}
func cloneReflect(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		out := reflect.New(v.Type()).Elem()
		out.Set(cloneReflect(v.Elem()))
		return out
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		out := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), cloneReflect(iter.Value()))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(cloneReflect(v.Index(i)))
		}
		return out
	default:
		return v
	}
}
