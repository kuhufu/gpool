# gpool

### example

```go
p := gpool.NewDefault()
p.Run(func(){
    fmt.Println("hello world")
})

p.Wait()
p.ShutdownDirectly()
```



```go
coreNum := runtime.NumCPU()
maxNUm := 100
bufLen := 20
p := gpool.New(coreNum, maxNum, bufLen, gpool.ByClan)

t := p.Run(func(){
    fmt.Println("hello world")
})

// 等待任务完成
<- t.Done()

//判断是否有错误发生
if t.Err() != nil {
    fmt.Println("error occur")    
}

p.ShutdownGracefully()
```

### Pool.Run和Pool.Call
Pool.Run方法的函数参数没有返回值
```go
p := gpool.NewDefault()

res := p.Run(func(){
    fmt.Println("hello world")
})

<-res.Done() //没有值，仅作为检查是否完成使用
```

Pool.Call方法的函数参数有返回值
```go
p := gpool.NewDefault()

res := p.Call(func() interface{} {
    fmt.Println("hello")
    return "world"
})

fmt.Println(<-res.Done()) //有值，为函数返回值，且只传递一次，
```

### gpool 的服务策略

当 gpool 无法立刻为新任务服务时，所采取的策略

1. `gpool.byChan` 

   等待任务缓冲区出现空闲

2. `gpool.byCaller`

   调用方自己执行该任务
  
3. `gpool.Discard`
    
    直接丢弃

### 注意

关闭 gpool实例后不要再使用该实例，否则会出现不可预料的结果