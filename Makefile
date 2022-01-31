all: tp1_broadcast_algorithm tp1_playground

tp1_broadcast_algorithm: main.go database.go menu.go message.go transaction.go
	mkdir -p dist
	go build -o dist/$@ $^

tp1_playground: play.go database.go message.go transaction.go
	mkdir -p dist
	go build -o dist/$@ $^

.PHONY: clean
clean:
	rm -f dist/tp1_playground dist/tp1_broadcast_algorithm
