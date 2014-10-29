include Makefile.inc

all : $(PROJECT)

$(PROJECT) : $(OBJS)
	$(GOC) $(FLAGS) -o $(PROJECT) $(OBJS) -Wl,-R,$(LIB_PATH)

distclean: clean
	rm Makefile.inc

clean:
	rm $(PROJECT)
