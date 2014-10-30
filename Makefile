include Makefile.inc

all : $(PROJECT)

parser: $(PARSER_SOURCES)
	$(GOC) $(FLAGS) -c -o parsers.o $(PARSER_SOURCES)
	$(OBJCOPY) -j .go_export parsers.o parsers.gox

utils: $(UTILS_SOURCES)
	$(GOC) $(FLAGS) -c -o utils.o $(UTILS_SOURCES)
	$(OBJCOPY) -j .go_export utils.o utils.gox

main: $(MAIN_SOURCES)
	$(GOC) $(FLAGS) -c -o main.o $(MAIN_SOURCES)

$(PROJECT): utils parser main
	$(GOC) $(FLAGS) -o $(PROJECT) main.o utils.o parsers.o -Wl,-R,$(LIB_PATH)

distclean: clean
	rm Makefile.inc

clean:
	rm -f $(PROJECT) parsers.o parsers.gox utils.o utils.gox main.o
