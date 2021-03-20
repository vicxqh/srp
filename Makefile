.PHONY : server all agent

server :
	@echo "building server..."
	go build -o ./bin/server ./server

agent :
	@echo "building agent..."
	go build -o ./bin/agent ./agent

all : server agent