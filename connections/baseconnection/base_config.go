package baseconnection

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"

	"gitee.com/dark.H/gs"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

type ProtocolConfig struct {
	Server       interface{} `json:"server"`
	ServerPort   int         `json:"server_port"`
	LocalPort    int         `json:"local_port"`
	LocalAddress string      `json:"local_address"`
	Password     string      `json:"password"`
	Method       string      `json:"method"` // encryption method
	Tag          string      `json:"tag"`

	// following options are only used by server
	PortPassword map[string]string `json:"port_password"`
	Timeout      int               `json:"timeout"`
	LastPing     int               `json:"last_ping"`
	// following options are only used by client

	// The order of servers in the client config is significant, so use array
	// instead of map to preserve the order.
	ServerPassword string `json:"server_password"`

	// shadowsocks options
	SSPassword  string `json:"ss_password"`
	OldSSPwd    string `json:"ss_old"`
	SSMethod    string `json:"ss_method"`
	SALT        string `json:"salt"`
	EBUFLEN     int    `json:"buflen"`
	Type        string `json:"type"`
	OtherConfig gs.Dict[any]
}

// GeneratePassword by config
func (config *ProtocolConfig) GeneratePassword(plugin ...string) (en kcp.BlockCrypt) {
	klen := 32
	if strings.Contains(config.Method, "128") {
		klen = 16
	}
	mainMethod := strings.Split(config.Method, "-")[0]
	var keyData []byte
	if config.SALT == "" && config.EBUFLEN == 0 {
		keyData = pbkdf2.Key([]byte(config.Password), []byte("demo salt"), 1024, klen, sha1.New)

		if plugin != nil {
			keyData = pbkdf2.Key([]byte(config.Password), []byte("kcp-go"), 4096, klen, sha1.New)
		}
	} else {
		keyData = pbkdf2.Key([]byte(config.Password), []byte(config.SALT), config.EBUFLEN, klen, sha1.New)
	}

	switch mainMethod {

	case "des":
		en, _ = kcp.NewTripleDESBlockCrypt(keyData[:klen])
	case "tea":
		en, _ = kcp.NewTEABlockCrypt(keyData[:klen])
	case "simple":
		en, _ = kcp.NewSimpleXORBlockCrypt(keyData[:klen])
	case "xtea":
		en, _ = kcp.NewXTEABlockCrypt(keyData[:klen])
	default:
		en, _ = kcp.NewAESBlockCrypt(keyData[:klen])
	}

	return
}

// GetServerArray get server
func (config *ProtocolConfig) GetServerArray() []string {
	// Specifying multiple servers in the "server" options is deprecated.
	// But for backward compatibility, keep this.
	if config.Server == nil {
		return nil
	}
	single, ok := config.Server.(string)
	if ok {
		return []string{single}
	}
	arr, ok := config.Server.([]interface{})
	if ok {
		serverArr := make([]string, len(arr), len(arr))
		for i, s := range arr {
			serverArr[i], ok = s.(string)
			if !ok {
				goto typeError
			}
		}
		return serverArr
	}
typeError:
	panic(fmt.Sprintf("Config.Server type error %v", reflect.TypeOf(config.Server)))
}

// ParseConfig parse path to json
func ParseConfig(path string) (config *ProtocolConfig, err error) {
	file, err := os.Open(path) // For read access.
	if err != nil {
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}

	config = &ProtocolConfig{}
	if err = json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	readTimeout = time.Duration(config.Timeout) * time.Second
	return
}
