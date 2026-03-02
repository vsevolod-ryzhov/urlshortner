#!/bin/zsh

run_test() {
    local test_num=$1
    local cmd=$2

    echo "Execution TestIteration${test_num}"
    eval $cmd

    if [ $? -eq 0 ]; then
        echo "TestIteration${test_num} passed"
    else
        echo "TestIteration${test_num} NOT passed"
        exit 1
    fi
}

go build -o cmd/shortener/shortener cmd/shortener/*.go || exit 1

go test ./... || exit 1

./cmd/staticlint/checker ./... || exit 1

run_test 1 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration1$ -binary-path=cmd/shortener/shortener"

run_test 2 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration2$ -source-path=."

run_test 3 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration3$ -source-path=."

run_test 4 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration4$ -binary-path=cmd/shortener/shortener -server-port=8088"

run_test 5 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration5$ -binary-path=cmd/shortener/shortener -server-port=8080"

run_test 6 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration6$ -source-path=."

run_test 7 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration7$ -binary-path=cmd/shortener/shortener -source-path=."

run_test 8 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration8$ -binary-path=cmd/shortener/shortener"

run_test 9 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration9$ -binary-path=cmd/shortener/shortener -source-path=. -file-storage-path=/tmp/someTmpFile"

run_test 10 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration10$ -binary-path=cmd/shortener/shortener -source-path=. -database-dsn=\"host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable\""

run_test 11 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration11$ -binary-path=cmd/shortener/shortener -source-path=. -database-dsn=\"host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable\""

run_test 12 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration12$ -binary-path=cmd/shortener/shortener -database-dsn=\"host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable\""

run_test 13 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration13$ -binary-path=cmd/shortener/shortener -database-dsn=\"host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable\""

run_test 14 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration14$ -binary-path=cmd/shortener/shortener -database-dsn=\"host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable\""

run_test 15 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration15$ -binary-path=cmd/shortener/shortener -database-dsn=\"host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable\""

run_test 16 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration16$ -source-path=."

run_test 17 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration17$ -source-path=."

run_test 18 "shortenertest_v2-darwin-arm64 -test.v -test.run=^TestIteration18$ -source-path=."

echo "All tests passed"