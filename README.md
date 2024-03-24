## 限流器
> 两种限流器：

1. [漏桶限流](/leaky.go), 可应对最大QPS场景
  - 请求的时间间隔需要大于PerRequest
  - 等待时间wait最大为PerRequest
2. [令牌桶限流器](/token.go), 可应对突发流量场景
  - 等待时间wait会累加

> 两种参数介质：

1. [内存存储](/atomic.go)，适用于单一主机；
2. [redis存储](/atomic_redis.go)，适用于分布式服务共用一个限流器，多个服务共享限流器参数(必须要相同的name)；缺点是要频繁访问redis，有很大网络消耗

### 使用
- using gin
```go
package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	lt "github.com/guanhg/ratelimit"
)

func main() {
	// 使用token限流器，每分钟最大100请求
	gl := lt.NewLimiter("MyLimiter", lt.Token, 100)
    
    /* 分布式服务使用redis介质存储参数，不同主机服务需要相同的name
    err := gl.SetRedis("127.0.0.1:6379")
	if err != nil {
		fmt.Errorf(err)
		return
	}
    */

	LimiteHandler := func() gin.HandlerFunc {
		return func(ctx *gin.Context) {
			wt, b, err := gl.Acquired()
			if err != nil {
				ctx.Abort()
				ctx.JSON(500, "Limiter error")
				return
			}
			if !b {
				ctx.Abort()
				waitResp := fmt.Sprintf("Server is busy! Wait for about %f secords", wt.Seconds())
				ctx.JSON(500, waitResp)
				return
			}
			ctx.Next()
		}
	}

	serv := gin.Default()
	serv.Use(LimiteHandler())
	serv.Any("/ping", func(c *gin.Context) {
		c.JSON(200, "Pong!")
	})

	serv.Run(":8080")
}

```


``` shell
# test
for ((i=1;i<100;i++)) do echo "$i: $(curl -s http://localhost:8080/ping) \n----"; sleep 0.05; done;
```