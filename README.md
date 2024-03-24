### 说明


### 使用


### 开发
>安装最新版 golang

### 安装bee
> 记得将`GOPATH/bin`添加到环境变量`PATH`，否则 `bee` 命令无法全局使用
> 使用 `go env GOPATH` 查看 `GOPATH` 路径

```
go install github.com/beego/bee/v2@latest
```

### 安装依赖

```
go mod tidy
```

### 启动
> http://localhost:8080

```
bee run
```

### 打包