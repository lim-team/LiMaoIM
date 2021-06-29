## 狸猫IM（LiMaoIM） 一切很简单

本项目是一款简单易用，性能强劲，设计理念简洁的即时通讯服务

### 特点

* go语言开发，高性能与易维护兼得。
* 100%开源，底裤都不留！
* 二进制协议(支持自定义)，包大小极小，最小心跳包只有1byte，耗电小，流量小，传输速度快。
* 消息通道和消息内容全程加密，防中间人攻击和串改消息内容。
* 采用tcp协议+ack机制保证消息稳定可靠不丢。
* 扩展性强 采用频道设计理念，目前支持群组频道，点对点频道，后续可以根据自己业务自定义频道可实现机器人频道，客服频道等等功能。
* 多端同步，web，pc，app消息实时同步。
* 同时无差别支持tcp，websocket。
* 万人群支持。
* 消息分区永久存储，卸载设备消息不丢。
* 支持读模式的离线拉取

### 快速入门

#### 安装

安装主要是获取到limaoim执行文件

***直接下载***

[Linux系统](https://baidu.com)

[Mac系统 ](https://baidu.com/)

注意：window暂不支持

#### 源码编译

```
 $ git clone https://github.com/lim-team/limaoim
 $ cd limaoim
 $ go build cmd/app/main.go -o limaoim
```

#### 运行(一键运行)

```
$ ./limaoim -c configs/config.toml
```

### 性能测试

一键压测

```
./bench.sh
```

本人测试结果如下：

达到每秒63420万条消息的吞吐量，接近redis的压测数据！

```
goos: darwin
goarch: amd64
pkg: github.com/lim-team/LiMaoIM/internal/lim
cpu: Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz

SEND: 2021/06/29 15:05:49 duration: 10.605478656s - 12.096mb/s - 63420.051ops/s - 15.768us/op
```


单机同时在线

### 快速试玩

***登录test用户***

```
// 登录test1
$ go run cmd/test/main.go login 127.0.0.1 -user=test1 -token=xxxx
```

```
// 登录test2
$ go run cmd/test/main.go login 127.0.0.1 -user=test2 -token=xxxx
```

test1监听消息
```
$ >watch
```

test2发送消息给test1

```
$ >send "this is test" -to test1
```


<!-- 
***分布式***

节点初始化

```
// 开启proxy服务 指定初始化的节点nodes
# limaoim proxy -c ./configs/proxy.toml  -e replica=1
```


```
// 初始化的节点启动
# limaoim -c ./configs/config.toml -proxy=xx.xx.xx.xx:16666 -e nodeID=1001 -e nodeAddr=127.0.0.1:6666
(或者 limaoim -c ./configs/config.toml -peers=1@http://127.0.0.1:6000,2@http://127.0.0.1:6001,3@http://127.0.0.1:6002 -e nodeID=1)
```

```
// 初始化的节点启动
# limaoim  -e proxy=xx.xx.xx.xx:16666 -e nodeID=1002 -e nodeAddr=127.0.0.1:6667
```

增加节点

```
# limaoim  -proxy=xx.xx.xx.xx:16666 -e nodeID=1003 -join
```

移除节点

```
# limaoim -e nodeID=1003 -remove
``` -->



#### 通过Docker Compose运行

```
$ docker-compose up 
```


### 文档

[LiMaoIM协议](./docs/protocol.md)

***架构***


### 案例

***截图***

***下载***