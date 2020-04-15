all: clean kernel8.img

uart: main.go
	tinygo build -target rpi3 -o uart main.go

kernel8.img: uart
	llvm-objcopy -O binary uart kernel8.img

clean:
	rm uart *.o >/dev/null 2>/dev/null || true

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio
