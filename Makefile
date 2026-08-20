.PHONY: lint
# 代码检查
lint:
	mise x golangci-lint@2.13.0 -- golangci-lint run -c .golangci.yml --allow-parallel-runners --timeout=10m
.PHONY: fmt
# 代码检查
fmt:
	mise x golangci-lint@2.13.0 -- golangci-lint fmt -v -c .golangci.yml
