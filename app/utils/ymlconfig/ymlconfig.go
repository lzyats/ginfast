package ymlconfig

import (
	"fmt"
	"gin-fast/app/global/app"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	secretclient "github.com/lzyats/core-secret-go/client"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var lastChangeTime time.Time

func init() {
	lastChangeTime = time.Now()
}

func CreateYamlFactory(path string, fileName ...string) app.YmlConfigInterf {
	yamlConfig := viper.New()
	yamlConfig.AddConfigPath(path)
	configName := "config"
	if len(fileName) == 0 {
		yamlConfig.SetConfigName(configName)
	} else {
		configName = fileName[0]
		yamlConfig.SetConfigName(configName)
	}
	yamlConfig.SetConfigType("yml")

	if err := yamlConfig.ReadInConfig(); err != nil {
		log.Fatal("ReadInConfig err: " + err.Error())
	}
	if err := decryptSecretConfig(yamlConfig); err != nil {
		log.Fatal("DecryptConfig err: " + err.Error())
	}

	return &ymlConfig{
		viper:      yamlConfig,
		mu:         new(sync.RWMutex),
		configPath: path,
		configName: configName,
		configType: "yml",
		pendingSet: make(map[string]interface{}),
	}
}

type secretBootstrapConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	ServerURL      string `mapstructure:"server_url"`
	AppID          string `mapstructure:"app_id"`
	AppSecret      string `mapstructure:"app_secret"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
	Batch          bool   `mapstructure:"batch"`
	Retry          int    `mapstructure:"retry"`
}

func decryptSecretConfig(cfg *viper.Viper) error {
	var secretCfg secretBootstrapConfig
	if err := cfg.UnmarshalKey("secret", &secretCfg); err != nil {
		return err
	}
	if !secretCfg.Enabled {
		return nil
	}

	appSecret := strings.TrimSpace(os.Getenv("APP_SECRET"))
	if appSecret == "" {
		appSecret = secretCfg.AppSecret
	}
	if secretCfg.AppID == "" || appSecret == "" {
		return fmt.Errorf("secret config incomplete: app_id/app_secret required when secret.enabled=true")
	}

	timeout := 5 * time.Second
	if secretCfg.TimeoutSeconds > 0 {
		timeout = time.Duration(secretCfg.TimeoutSeconds) * time.Second
	}

	settings := cfg.AllSettings()
	decryptor := secretclient.New(secretclient.Options{
		ServerURL: secretCfg.ServerURL,
		AppID:     secretCfg.AppID,
		AppSecret: appSecret,
		Timeout:   timeout,
		Batch:     secretCfg.Batch,
		Retry:     secretCfg.Retry,
	})
	decrypted, err := decryptSettings(settings, "", decryptor)
	if err != nil {
		return err
	}
	applySettings(cfg, "", decrypted)
	return nil
}

func applySettings(cfg *viper.Viper, prefix string, value interface{}) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			applySettings(cfg, next, item)
		}
	case map[interface{}]interface{}:
		for rawKey, item := range typed {
			key := fmt.Sprint(rawKey)
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			applySettings(cfg, next, item)
		}
	default:
		if prefix != "" {
			cfg.Set(prefix, value)
		}
	}
}

func decryptSettings(value interface{}, scene string, decryptor *secretclient.Client) (interface{}, error) {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			nextScene := key
			if scene != "" {
				nextScene = scene + "." + key
			}
			decrypted, err := decryptSettings(item, nextScene, decryptor)
			if err != nil {
				return nil, err
			}
			result[key] = decrypted
		}
		return result, nil
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{}, len(typed))
		for rawKey, item := range typed {
			key := fmt.Sprint(rawKey)
			nextScene := key
			if scene != "" {
				nextScene = scene + "." + key
			}
			decrypted, err := decryptSettings(item, nextScene, decryptor)
			if err != nil {
				return nil, err
			}
			result[rawKey] = decrypted
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(typed))
		for i, item := range typed {
			nextScene := scene
			if nextScene != "" {
				nextScene = nextScene + "." + strconv.Itoa(i)
			}
			decrypted, err := decryptSettings(item, nextScene, decryptor)
			if err != nil {
				return nil, err
			}
			result[i] = decrypted
		}
		return result, nil
	case string:
		return decryptor.DecryptIfEncrypted(typed, scene)
	default:
		return value, nil
	}
}

type ymlConfig struct {
	viper      *viper.Viper
	mu         *sync.RWMutex
	configPath string
	configName string
	configType string
	pendingSet map[string]interface{}
}

func (y *ymlConfig) ConfigFileChangeListen(fns ...func()) {
	y.viper.OnConfigChange(func(changeEvent fsnotify.Event) {
		if time.Since(lastChangeTime).Seconds() >= 1 && changeEvent.Op.String() == "WRITE" {
			y.mu.Lock()
			if err := y.viper.ReadInConfig(); err != nil {
				log.Printf("重新读取配置文件失败: %v", err)
			} else if err := decryptSecretConfig(y.viper); err != nil {
				log.Printf("配置文件解密失败: %v", err)
			} else {
				log.Println("配置文件重新加载成功")
			}
			y.mu.Unlock()

			for _, f := range fns {
				f()
			}
			lastChangeTime = time.Now()
		}
	})
	y.viper.WatchConfig()
}

func (y *ymlConfig) Get(keyName string) interface{} {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.Get(keyName)
}

func (y *ymlConfig) GetString(keyName string) string {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetString(keyName)
}

func (y *ymlConfig) GetBool(keyName string) bool {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetBool(keyName)
}

func (y *ymlConfig) GetInt(keyName string) int {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetInt(keyName)
}

func (y *ymlConfig) GetInt32(keyName string) int32 {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetInt32(keyName)
}

func (y *ymlConfig) GetInt64(keyName string) int64 {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetInt64(keyName)
}

func (y *ymlConfig) GetFloat64(keyName string) float64 {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetFloat64(keyName)
}

func (y *ymlConfig) GetDuration(keyName string) time.Duration {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetDuration(keyName)
}

func (y *ymlConfig) GetStringSlice(keyName string) []string {
	y.mu.RLock()
	defer y.mu.RUnlock()
	return y.viper.GetStringSlice(keyName)
}

func (y *ymlConfig) GetUintSlice(keyName string) []uint {
	y.mu.RLock()
	defer y.mu.RUnlock()

	if value := y.viper.Get(keyName); value != nil {
		if uintSlice, ok := value.([]uint); ok {
			return uintSlice
		}
	}

	intSlice := y.viper.GetIntSlice(keyName)
	if len(intSlice) == 0 {
		return []uint{}
	}

	uintSlice := make([]uint, len(intSlice))
	for i, v := range intSlice {
		if v < 0 {
			uintSlice[i] = 0
		} else {
			uintSlice[i] = uint(v)
		}
	}

	return uintSlice
}

func (y *ymlConfig) Set(keyName string, value interface{}) {
	y.mu.Lock()
	defer y.mu.Unlock()
	y.viper.Set(keyName, value)
	y.pendingSet[keyName] = value
}

func (y *ymlConfig) SaveConfig() error {
	y.mu.Lock()
	defer y.mu.Unlock()

	if len(y.pendingSet) == 0 {
		return nil
	}

	configFile := filepath.Join(y.configPath, y.configName+"."+y.configType)
	content, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return err
	}

	for key, value := range y.pendingSet {
		if err := setYAMLValue(&root, strings.Split(key, "."), value); err != nil {
			return err
		}
	}

	updated, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configFile, updated, 0644); err != nil {
		return err
	}

	y.pendingSet = make(map[string]interface{})
	return nil
}

func setYAMLValue(root *yaml.Node, path []string, value interface{}) error {
	if len(root.Content) == 0 {
		root.Kind = yaml.DocumentNode
		root.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}

	return setYAMLMappingValue(root.Content[0], path, value)
}

func setYAMLMappingValue(node *yaml.Node, path []string, value interface{}) error {
	if len(path) == 0 {
		return nil
	}

	if node.Kind == 0 {
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("yaml path %s is not a mapping node", strings.Join(path, "."))
	}

	key := path[0]
	valueNode := findYAMLMapValue(node, key)
	if valueNode == nil {
		valueNode = &yaml.Node{}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			valueNode,
		)
	}

	if len(path) == 1 {
		assignYAMLNode(valueNode, value)
		return nil
	}

	if valueNode.Kind == 0 {
		valueNode.Kind = yaml.MappingNode
		valueNode.Tag = "!!map"
	}

	return setYAMLMappingValue(valueNode, path[1:], value)
}

func findYAMLMapValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func assignYAMLNode(dst *yaml.Node, value interface{}) {
	headComment := dst.HeadComment
	lineComment := dst.LineComment
	footComment := dst.FootComment
	style := dst.Style

	newNode := buildYAMLNode(value)
	*dst = *newNode
	dst.HeadComment = headComment
	dst.LineComment = lineComment
	dst.FootComment = footComment
	dst.Style = style
}

func buildYAMLNode(value interface{}) *yaml.Node {
	switch typed := value.(type) {
	case map[string]interface{}:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for key, item := range typed {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
				buildYAMLNode(item),
			)
		}
		return node
	case []string:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			node.Content = append(node.Content, buildYAMLNode(item))
		}
		return node
	case []int:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			node.Content = append(node.Content, buildYAMLNode(item))
		}
		return node
	case []uint:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			node.Content = append(node.Content, buildYAMLNode(item))
		}
		return node
	case []interface{}:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range typed {
			node.Content = append(node.Content, buildYAMLNode(item))
		}
		return node
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: typed}
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(typed)}
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(typed)}
	case int32:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(int64(typed), 10)}
	case int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(typed, 10)}
	case uint:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(uint64(typed), 10)}
	case uint32:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(uint64(typed), 10)}
	case uint64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(typed, 10)}
	case float32:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(float64(typed), 'f', -1, 32)}
	case float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(typed, 'f', -1, 64)}
	default:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprint(value)}
	}
}
