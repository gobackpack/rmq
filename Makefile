COVERAGE_FILE="coverage_file"

test-cover:
	go test -v ./... -coverprofile=${COVERAGE_FILE}
	go tool cover -html=${COVERAGE_FILE} && go tool cover -func ${COVERAGE_FILE} && unlink ${COVERAGE_FILE}