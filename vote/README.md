# Leader Election
Log는 제외하고 RequestVote 위주로 리더 선출 구현. AppendEntries 요청은 term이 낮지 않으면 무조건 성공.

## Run
클러스터 구성 세팅에서 Peers를 고정으로 해놓았기 때문에 flag 설정을 변경하면 안됨. `main.go: Peers`에 고정해놓음.

- id: 1
```
$ go run main.go --id=1 --addr=localhost:10000
```

- id: 2
```
$ go run main.go --id=2 --addr=localhost:10001
```

- id: 3
```
$ go run main.go --id=3 --addr=localhost:10002
```

## Result
### 1. 첫번째 리더 선출
log entries 상황에 따라 RequestVote 요청을 거부하는 로직이 없어서 약간의 버그가 있지만 id=1 인 서버가 term=4 으로 리더에 선출됨.

- id: 1
```
2021/02/20 15:10:31 Election timeout!
2021/02/20 15:10:31 StartElection: Start Election with Term:2
2021/02/20 15:10:31 StartElection: RequestVote to 2 fail: rpc error: code = DeadlineExceeded desc = context deadline exceeded
2021/02/20 15:10:31 StartElection: RequestVote to 3 fail: rpc error: code = DeadlineExceeded desc = context deadline exceeded
2021/02/20 15:10:32 Election timeout!
2021/02/20 15:10:32 StartElection: Start Election with Term:3
2021/02/20 15:10:32 StartElection: RequestVote to 3 fail: rpc error: code = DeadlineExceeded desc = context deadline exceeded
2021/02/20 15:10:32 StartElection: RequestVote to 2 fail: rpc error: code = DeadlineExceeded desc = context deadline exceeded
2021/02/20 15:10:32 Election timeout!
2021/02/20 15:10:32 StartElection: Start Election with Term:4
2021/02/20 15:10:32 StartElection: Win election. Start leader # 리더 선출!
2021/02/20 15:10:32 StartElection: RequestVote to 3 fail: rpc error: code = Canceled desc = context canceled
2021/02/20 15:10:32 StartLeader: Send heartbeat
2021/02/20 15:10:32 StartLeader: Send heartbeat
```

- id: 2
```
2021/02/20 15:10:31 RequestVote: There is other leader that term is 2
2021/02/20 15:10:32 RequestVote: There is other leader that term is 3
2021/02/20 15:10:32 RequestVote: There is other leader that term is 4
2021/02/20 15:10:32 RequestVote: vote for 1 in term 2
2021/02/20 15:10:32 RequestVote: vote for 1 in term 3
2021/02/20 15:10:32 RequestVote: vote for 1 in term 4
2021/02/20 15:10:32 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:32 AppendEntries: Receive AppendEntries Request in term 4
```

- id: 3
이미 id=1 서버가 term=4 에서 리더로 선출됨.
- 버그
  - term이 낮은 상태로 AppendEntries를 받으면 해당 term의 follower로 바로 변경되어야 하는데 term 업데이트되지 않은 상태로 여러번 받고있음.
  - RequestVote, AppendEntries grpc 에는 context를 다른 고루틴과 맞추질 못해서 서순 오류가 있다.
```
2021/02/20 15:10:33 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:33 AppendEntries: There is other leader that term is 4
2021/02/20 15:10:33 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:33 AppendEntries: There is other leader that term is 4
2021/02/20 15:10:33 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:33 AppendEntries: There is other leader that term is 4
2021/02/20 15:10:33 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:33 AppendEntries: There is other leader that term is 4
2021/02/20 15:10:33 RequestVote: vote for 1 in term 3
2021/02/20 15:10:33 RequestVote: vote for 1 in term 4
2021/02/20 15:10:33 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:33 AppendEntries: Receive AppendEntries Request in term 4
```

### 2. 리더 강제종료, 새로운 리서 선출
id=1 서버가 종료된 후 id=2 서버가 먼저 timeout 발생. id=3 의 투표를 받고 과반수 2표를 받음. term=5 으로 리더 선출. 
- id: 1
```
2021/02/20 15:10:37 StartLeader: Send heartbeat
2021/02/20 15:10:37 StartLeader: Send heartbeat
2021/02/20 15:10:37 StartLeader: Send heartbeat
^Csignal: interrupt
```

- id: 2
```
2021/02/20 15:10:37 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:37 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:38 Election timeout!
2021/02/20 15:10:38 StartElection: Start Election with Term:5
2021/02/20 15:10:38 StartElection: Win election. Start leader # 리더 선출
2021/02/20 15:10:38 StartElection: RequestVote to 1 fail: rpc error: code = Canceled desc = latest balancer error: connection error: desc = "transport: Error while dialing dial tcp 127.0.0.1:10000: connect: connection refused"
2021/02/20 15:10:38 StartLeader: Send heartbeat
2021/02/20 15:10:38 StartLeader: Send heartbeat
```

- id: 3
```
2021/02/20 15:10:37 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:37 AppendEntries: Receive AppendEntries Request in term 4
2021/02/20 15:10:38 RequestVote: There is other leader that term is 5
2021/02/20 15:10:38 RequestVote: vote for 2 in term 5
2021/02/20 15:10:38 AppendEntries: Receive AppendEntries Request in term 5
2021/02/20 15:10:38 AppendEntries: Receive AppendEntries Request in term 5
```
