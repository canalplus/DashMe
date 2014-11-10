include Makefile.inc

FFMPEGC_SOURCES = $(OBJDIR)/_cgo_defun.o $(OBJDIR)/_cgo_export.o $(OBJDIR)/src_parser_ffmpeg.cgo2.o
FFMPEGGO_SOURCES = $(OBJDIR)/_cgo_gotypes.go $(OBJDIR)/src_parser_ffmpeg.cgo1.go

all : $(PROJECT)

ffmpeg: $(FFMPEG_SOURCES)
	$(CGO) -gccgo=true -objdir=$(OBJDIR) $(FFMPEG_SOURCES)

$(OBJDIR)/parser.o: $(PARSER_SOURCES) $(FFMPEGGO_SOURCES)
	$(GOC) $(FLAGS) -c -o $(OBJDIR)/parser.o $(PARSER_SOURCES) $(FFMPEGGO_SOURCES)

$(OBJDIR)/utils.o: $(UTILS_SOURCES)
	$(GOC) $(FLAGS) -c -o $(OBJDIR)/utils.o $(UTILS_SOURCES)

parsers: $(OBJDIR)/parser.o
	$(OBJCOPY) -j .go_export $(OBJDIR)/parser.o parser.gox

utils: $(OBJDIR)/utils.o
	$(OBJCOPY) -j .go_export $(OBJDIR)/utils.o utils.gox

main: $(MAIN_SOURCES)
	$(GOC) $(FLAGS) -c -o $(OBJDIR)/main.o $(MAIN_SOURCES)

$(PROJECT): ffmpeg $(FFMPEGC_SOURCES) utils parsers main
	$(GOC) $(FLAGS) -o $(PROJECT) $(OBJDIR)/main.o $(OBJDIR)/utils.o $(OBJDIR)/parser.o $(FFMPEGC_SOURCES) -Wl,-R,$(LIB_PATH) $(LIBS)

distclean: clean
	rm -rf Makefile.inc $(OBJDIR)

%.o:%.c
	$(CC) $(CFLAGS) -c $< -o $@

clean:
	rm -rf $(PROJECT) $(OBJDIR)/* parser.gox utils.gox

.PHONY: doc
