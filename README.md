# gpool

### example

```go
p := gpool.NewDefault()
p.Run(func(){
    fmt.Println("hello world")
})

p.Wait()
p.Close()
```



```go
coreNum := runtime.NumCPU()
maxNUm := 100
bufLen := 20
p := gpool.New(coreNum, maxNum, bufLen, gpool.ByClan)

p.Run(func(){
    fmt.Println("hello world")
})

p.WaitAndClose()
```

### gpool 的服务策略

当 gpool 无法立刻为新任务服务时，所采取的策略

1. `gpool.byChan` 

   等待任务缓冲区出现空闲

2. `gpool.byCaller`

   调用方自己执行该任务

### 注意

关闭 gpool实例后不要再使用该实例，否则会出现不可预料的结果