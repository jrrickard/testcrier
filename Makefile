default:
	mkdir -p dist
	go get ./... github.com/jrrickard/testcrier/
	go build -o dist/testcrier github.com/jrrickard/testcrier
	GOOS=linux GOARCH=amd64 go build -o dist/testcrier-linux github.com/jrrickard/testcrier

test:
	go test github.com/jrrickard/testcrier/ 

fmt:
	go fmt github.com/jrrickard/testcrier/

clean:
	go clean github.com/jrrickard/testcrier

