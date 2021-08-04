# Calldiff

## 编译

```bash
go build -o calldiff main.go
```

## 运行

```bash
./calldiff -url=https://github.com/*.git -dir=/path/to/repo -old=<Commit ID> -new=<Commit ID> /path/to/repo/package
```

## 参数

### 主要参数

- **url** : 项目的 Git 地址，如果已经 clone 到本地的话，则不再需要额外指定 url 。
- **dir** : 项目的本地绝对路径
- **old** : 需要对比的，旧的 Commit ID
- **new** : 需要对比的，新的 Commit ID

**在参数的最后，必须加上需要分析的 package**

### 其他参数

更多的参数可以通过：

```bash
./calldiff -h
```

进一步了解。