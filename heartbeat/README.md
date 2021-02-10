# Heartbeats

- gRPC 엔드포인트 리스트를 관리
- 지속적으로 엔드포인트에 RPC 호출
- 만약 새로운 ID로 요청이 오면 엔드포인트에 추가

## Run
- terminal 1
```
$ go run main.go --id=1 --host=localhost:10000 --discover=nil
```

- terminal 2
```
$ go run main.go --id=2 --host=localhost:10001 --discover=localhost:10000
```

- terminal 3
```
$ go run main.go --id=3 --host=localhost:10002 --discover=localhost:10000
```

## Result
- terminal 1

![image](https://user-images.githubusercontent.com/44857109/107532559-5f071980-6c01-11eb-875b-a954090c804f.png)

- terminal 2

![image](https://user-images.githubusercontent.com/44857109/107532688-7fcf6f00-6c01-11eb-83ec-9d030df485db.png)

- terminal 3

![image](https://user-images.githubusercontent.com/44857109/107532730-8b229a80-6c01-11eb-8206-d1f567b3986b.png)
