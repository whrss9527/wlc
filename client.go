package wlc

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Option func(c *client)

func WithHTTPClient(c *http.Client) Option {
	return func(cc *client) {
		if c != nil {
			cc.client = c
		}
	}
}

type client struct {
	appId        string
	secretKey    string
	secretKeyHex []byte
	bizId        string
	client       *http.Client
}

// Client 生产客户端
type Client interface {
	// Check 实名认证接口,
	// 网络游戏用户实名认证服务接口，面向已经接入网络游戏防
	// 沉迷实名认证系统的游戏运营单位提供服务，游戏运营单位调用
	// 该接口进行用户实名认证工作，本版本仅支持大陆地区的姓名和
	// 二代身份证号核实认证。
	Check(ctx context.Context, param CheckParam) (*CheckResult, error)

	// Query 实名认证结果查询接口,
	// 网络游戏用户实名认证结果查询服务接口，面向已经提交用
	// 户实名认证且没有返回结果的游戏运营单位提供服务，游戏运营
	// 单位可以调用该接口，查询已经提交但未返回结果用户的实名认
	// 证结果。
	Query(ctx context.Context, ai string) (*QueryResult, error)

	// LoginTrace 游戏用户行为数据上报接口
	// 游戏用户行为数据上报接口，面向已经接入网络游戏防沉迷
	// 实名认证系统的游戏运营单位提供服务，游戏运营单位调用该接
	// 口上报游戏用户上下线行为数据。
	LoginTrace(ctx context.Context, param LoginTraceParam) ([]*LoginTraceResult, error)
}

// TestClient 接口测试辅助客户端
type TestClient interface {
	CheckTest(ctx context.Context, code string, param CheckParam) (*CheckResult, error)

	QueryTest(ctx context.Context, code, ai string) (*QueryResult, error)

	LoginTraceTest(ctx context.Context, code string, param LoginTraceParam) ([]*LoginTraceResult, error)
}

func New(appId, secretKey, bizId string, opts ...Option) Client {
	var nClient = &client{}
	nClient.appId = appId
	nClient.secretKey = secretKey
	nClient.secretKeyHex, _ = hex.DecodeString(secretKey)
	nClient.bizId = bizId
	nClient.client = http.DefaultClient
	for _, opt := range opts {
		if opt != nil {
			opt(nClient)
		}
	}
	return nClient
}

func NewTest(appId, secretKey, bizId string, opts ...Option) TestClient {
	var nClient = &client{}
	nClient.appId = appId
	nClient.secretKey = secretKey
	nClient.secretKeyHex, _ = hex.DecodeString(secretKey)
	nClient.bizId = bizId
	nClient.client = http.DefaultClient
	for _, opt := range opts {
		if opt != nil {
			opt(nClient)
		}
	}
	return nClient
}

func (c *client) request(ctx context.Context, method, api string, values url.Values, param, result interface{}) error {
	if values == nil {
		values = url.Values{}
	}

	var body string
	if param != nil {
		data, err := json.Marshal(param)
		if err != nil {
			return err
		}

		// 加密请求参数
		if data, err = c.encrypt(c.secretKeyHex, data); err != nil {
			return err
		}

		// 构造新的请求参数
		var payload = struct {
			Data string `json:"data"`
		}{
			Data: base64.StdEncoding.EncodeToString(data),
		}

		if data, err = json.Marshal(payload); err != nil {
			return err
		}

		body = string(data)
	}

	var nURL = api
	if len(values) > 0 {
		nURL = api + "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, nURL, strings.NewReader(body))
	if err != nil {
		return err
	}

	var now = strconv.FormatInt(time.Now().UnixNano()/1e6, 10)

	values.Add("appId", c.appId)
	values.Add("bizId", c.bizId)
	values.Add("timestamps", now)

	var sign = c.sign(c.secretKey, values, body)

	req.Header.Set("appId", c.appId)
	req.Header.Set("bizId", c.bizId)
	req.Header.Set("timestamps", now)
	req.Header.Set("sign", sign)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	rsp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	if err = json.NewDecoder(rsp.Body).Decode(result); err != nil {
		return err
	}

	return nil
}

func (c *client) encrypt(secretKeyHex []byte, data []byte) ([]byte, error) {
	var block, err = aes.NewCipher(secretKeyHex)
	if err != nil {
		return nil, err
	}

	mode, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	var nonce = make([]byte, mode.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return mode.Seal(nonce, nonce, data, nil), nil
}

func (c *client) sign(secretKey string, values url.Values, body string) string {
	var pList = make([]string, 0, 3+len(values))

	for key := range values {
		pList = append(pList, key+values.Get(key))
	}
	sort.Strings(pList)
	var data = secretKey + strings.Join(pList, "") + body

	var h = sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
