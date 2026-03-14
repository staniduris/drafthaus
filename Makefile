.PHONY: build run test clean demo

build:
	go build -o drafthaus ./cmd/drafthaus/

run: build
	./drafthaus serve demo.draft

test:
	go test ./... -count=1

clean:
	rm -f drafthaus *.draft *.draft-wal *.draft-shm

demo: build clean
	./drafthaus init demo --template blog
	@echo ""
	@echo "Run: ./drafthaus serve demo.draft"
	@echo "Site: http://localhost:3000"
	@echo "Admin: http://localhost:3000/_admin (admin/admin)"
