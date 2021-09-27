## 网络游戏防沉迷实名认证系统(实名认证)

[https://wlc.nppa.gov.cn/fcm_company/index.html](https://wlc.nppa.gov.cn/fcm_company/index.html)

### 初始化

```go
import "github.com/smartwalle/wlc"

var client = wlc.New("app id", "secret key", "biz id")

```

### 实名认证

```go
client.Check()
```

### 认证结果查询

```go
client.Query()
```

### 上报用户行为数据

```go
client.LoginTrace()
```

## 注意

cmd 目录中 main 用于完成《网络游戏防沉迷实名认证系统》中的接口测试，该测试和实际需要用到的接口不是同一套。