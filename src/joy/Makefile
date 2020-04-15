all: clean joy

joy: *.go
	tinygo build -target rpi3 -ldflags='-T ../../tinygo_delta/kernel.ld' -o joy .

clean:
	rm joy *.o >/dev/null 2>/dev/null || true

