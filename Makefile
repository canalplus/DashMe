include Makefile.inc

FFMPEGC_SOURCES = $(INTEROPDIR)/_cgo_defun.o $(INTEROPDIR)/_cgo_export.o $(INTEROPDIR)/src_parser_ffmpeg.cgo2.o
FFMPEGGO_SOURCES = $(INTEROPDIR)/_cgo_gotypes.go $(INTEROPDIR)/src_parser_ffmpeg.cgo1.go

all : $(PROJECT)

ffmpeg: $(FFMPEG_SOURCES)
	$(CGO) -gccgo=true -objdir=$(INTEROPDIR) $(FFMPEG_SOURCES)

parser.o: $(PARSER_SOURCES) $(FFMPEGGO_SOURCES)
	$(GOC) $(FLAGS) -c -o parser.o $(PARSER_SOURCES) $(FFMPEGGO_SOURCES)

utils.o: $(UTILS_SOURCES)
	$(GOC) $(FLAGS) -c -o utils.o $(UTILS_SOURCES)

parsers: parser.o
	$(OBJCOPY) -j .go_export parser.o parser.gox

utils: utils.o
	$(OBJCOPY) -j .go_export utils.o utils.gox

main: $(MAIN_SOURCES)
	$(GOC) $(FLAGS) -c -o main.o $(MAIN_SOURCES)

$(PROJECT): ffmpeg $(FFMPEGC_SOURCES) utils parsers main
	$(GOC) $(FLAGS) -o $(PROJECT) main.o utils.o parser.o $(FFMPEGC_SOURCES) -Wl,-R,$(LIB_PATH) -lavformat -lavutil

distclean: clean
	rm -rf Makefile.inc $(INTEROPDIR)

%.o:%.c
	$(CC) $(CFLAGS) -c $< -o $@

clean:
	rm -rf $(PROJECT) parser.o parser.gox utils.o utils.gox main.o
