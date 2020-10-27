## vmess2json 工具

一个用于从v2ray订阅链接生成json片段的工具.

## 编译

安装golang/git环境后, 在项目目录执行以下命令编译
```
go build
```

即可完成编译，编译生成vmess2json可执行程序.


## 使用方法

vmess2json使用示例
```
./vmess2json --subscribe https://feed.v2ray.example/
```

将输出v2ray的Url集合，以及v2ray的config.json中outbounds数组json片段，复制json片段到outbounds数组中然后重启v2ray服务即可使用.