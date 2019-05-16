### Benchmarks

##### Node.js

```
⋊> ~/w/go-jsondiffpatch on master ⨯ make benchmark-node
node benchmarks/node/main.js
Tries: 100
Time: 1190951 (micro secs)
```

##### Single thread

```
⋊> ~/w/go-jsondiffpatch on master ⨯ make benchmark-go
./benchmark-go
Tries: 100
Time: 311535 (micro secs)
```

##### Mutexes

```
⋊> ~/w/go-jsondiffpatch on feature/shared-mem ⨯ make benchmark-go
./benchmark-go
Tries: 100
Time: 557361 (micro secs)
```

##### Channels

```
⋊> ~/w/go-jsondiffpatch on feature/channels ⨯ make benchmark-go
./benchmark-go
Tries: 100
Time: 846924 (micro secs)
```