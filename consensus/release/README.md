# Release Contract

## Prepare

```bash
# npm i solc@0.8.3 -g
# solcjs --version
0.8.3+commit.8d00100c.Emscripten.clang

# solcjs --optimize --abi --bin ../release/mapping.sol

# go install github.com/ethereum/go-ethereum/cmd/abigen@v1.10.10

# abigen -v
abigen version 1.10.10-stable

# abigen --abi release.abi --pkg release --type Token -out release.go
```
