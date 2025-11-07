## Tests
- go test ./...
1. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration1$ -binary-path=cmd/shortener/shortener
2. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration2$ -source-path=.
3. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration3$ -source-path=.
4. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration4$ -binary-path=cmd/shortener/shortener -server-port=8088
5. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration5$ -binary-path=cmd/shortener/shortener -server-port=8080
6. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration6$ -source-path=.
7. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration7$ -binary-path=cmd/shortener/shortener -source-path=.
8. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration8$ -binary-path=cmd/shortener/shortener
9. shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration9$ -binary-path=cmd/shortener/shortener -source-path=. -file-storage-path=/tmp/someTmpFile

## Misc
- go build -o cmd/shortener/shortener cmd/shortener/*.go
- lsof -nP -i4TCP:8888 | grep LISTEN
- curl -X POST -d "url=https://ya.ru" 127.0.0.1:8888 -v