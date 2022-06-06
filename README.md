## 限流器
- 三种限流器：
1. [计数限流器](/counter.go), 可应对限制爬虫场景
2. [漏桶限流](/leaky.go), 可应对最大QPS场景
3. [令牌桶限流器](/token.go), 可应对突发流量场景
- 两种参数存储：
1. [内存存储](/atomic.go)，适用于单一服务；如果在网关层使用，也可以适用多个服务；
2. [redis存储](/atomic_redis.go)，适用于某些特定多服务场景，多个服务共享限流器参数；(缺点是要频繁访问redis，有很大网络消耗)

### 使用
- using gin
```go
package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	limiter "github.com/guanhg/ratelimit"
)

func main() {
	ginLimiter := limiter.New("MyLimiter").  // 设置限流器名称
        CounterType().  // 这是计数限流器类型
        ProcessStorage(). // 设置普通内存模式
        Unit(time.Second). // 设置单位时间
        MaxReq(10). // 设置单位时间内最大请求数(最大漏水或最大令牌数)
        Build(limiter.WithIns(10)) // 生成限流器,为了使限流器平滑,带可选参数
    
    /* 使用redis存储参数
    ginLimiter := limiter.New("MyProcCounterLimiter").
        CounterType().
        RedisStorage("127.0.0.1:6379"). // 设置redis存储模式
        Unit(time.Second).
        MaxReq(10).
        Build(limiter.WithIns(10))
    */

	Limiter := func() gin.HandlerFunc {
		return func(ctx *gin.Context) {
			wt, b, err := ginLimiter.Release()
			if err != nil {
				ctx.Abort()
				ctx.JSON(500, "ginLimiter error")
				return
			}
			if !b {
				ctx.Abort()
				waitResp := fmt.Sprintf("Ops! Server is busy! Wait for about %f secords", wt.Seconds())
				ctx.JSON(500, waitResp)
				return
			}
			ctx.Next()
		}
	}

	serv := gin.Default()
	serv.Use(Limiter())
	serv.Any("/ping", func(c *gin.Context) {
		c.JSON(200, "Pong!")
	})

	serv.Run(":8080")
}

```


``` shell
	# test
    for ((i=1;i<100;i++)) do echo "$i: $(curl -s http://localhost:8080/ping) \n----"; sleep 0.05 done;
```