TINYGO=tinygo

P=../../src/examples/blinky1

stm32f=\
	stm32f469disco\
	bluepill\
	feather-stm32f405\
	mksnanov3\
	nucleo-f103rb\
	nucleo-f722ze\

stm32wl=\
	nucleo-wl55jc\
	gnse\
	lorae5\

stm32g=\
	nucleo-g031k8\
	nucleo-g071rb\

stm32h=\
	nucleo-h723zg\

stm32l=\
	nucleo-l031k6\
	nucleo-l432kc\
	nucleo-l476rg\
	nucleo-l552ze\
	swan\

targets=\
	$(stm32f)\
	$(stm32g)\
	$(stm32h)\
	$(stm32l)\
	$(stm32wl)\

all: test-build

test-build: $(foreach t,$(targets),build_$t.status)

build_%.status:
	@if $(TINYGO) build -o b.$*.elf -target $* $P 2> build_$*.log; then \
		echo build $* ok; \
		> $@; \
		rm -f b.$*.elf; \
	else \
		echo build $* failed; \
		echo FAILED > $@; \
		echo >> $@; \
		cat build_$*.log >> $@; \
	fi
	@rm -f build_$*.log \

clean:
	rm -f build_*.status
	rm -f b.*.elf
	rm -f build_*log
