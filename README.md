## 限流器
> 三种限流器：

1. [漏桶限流](./leaky.go), 可应对最大QPS场景
  - 请求的时间间隔需要大于PerRequest
  - 等待时间最大为PerRequest
2. [令牌桶限流器](./token.go), 可应对突发流量场景
  - 等待时间会累加
3. [计数限流器](./count.go)

### 使用
- gin
```go
package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	lt "github.com/guanhg/ratelimit"
)

func main() {
	// 使用token限流器，每秒最大10请求
	gl := lt.NewTokenLimiter(10)

	LimiterHandler := func() gin.HandlerFunc {
		return func(ctx *gin.Context) {
			wt, b := gl.Take()
			if !b {
				ctx.Abort()
				waitResp := fmt.Sprintf("Server is busy! Wait for about %02f secords", wt.Seconds())
				ctx.JSON(500, waitResp)
				return
			}
			ctx.Next()
		}
	}

	serv := gin.Default()
	serv.Use(LimiterHandler())
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