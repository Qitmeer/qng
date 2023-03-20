# Wallet

### First Create Or Import a account

```
~ ./qng --privnet -A=./ account import (privateKey path)
or
~ ./qng --privnet -A=./ account new
```

### Unlock account

```
~ ./cli.sh unlock 0 123456 60
or 
~
curl -k -u test:test -k -X POST   https://127.0.0.1:38131/   -H 'content-type: application/json'   -d '{
  "method":"wallet_unlock",
  "version":"2.0",
  "params":["0","123456",600],                                                                                                  "id":1
}'
```

### lock account

```
~ ./cli.sh lock Rk8Jwy4QbiK84iLDLFn81NzedVwFE7UWJqfWy9hf9GTWVGr3hmJk3
or 
curl -k -u test:test -k -X POST   https://127.0.0.1:38131/   -H 'content-type: application/json'   -d '{
  "method":"wallet_lock",
  "version":"2.0",
  "params":["Rk8Jwy4QbiK84iLDLFn81NzedVwFE7UWJqfWy9hf9GTWVGr3hmJk3"],
  "id":1
}'
```

### send to address

```
~ ./cli.sh sendtoaddress Rk8Jwy4QbiK84iLDLFn81NzedVwFE7UWJqfWy9hf9GTWVGr3hmJk3 "{\\\"RmN6q2ZdNaCtgpq2BE5ZaUbfQxXwRU1yTYf\\\":{\\\"amount\\\":100000000,\\\"coinid\\\":0}}" 0
or 
curl -u test:test -k -X POST   https://127.0.0.1:38131/   -H 'cache-control: no-cache'   -H 'content-type: application/json'   -H 'postman-token: 439c084c-4898-c548-cc7e-3121cfaea8f8'   -d '{
  "method":"wallet_sendToAddress",
  "version":"2.0",
  "params":["RmN6q2ZdNaCtgpq2BE5ZaUbfQxXwRU1yTYf","{\"RmN6q2ZdNaCtgpq2BE5ZaUbfQxXwRU1yTYf\":{\"amount\":100000000,\"coinid\":0}}",0],
  "id":1
}'
```





