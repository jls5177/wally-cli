package main

import (
	"flag"
	"fmt"
	wallycli "github.com/jls5177/wally-cli"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/spin"
	"gopkg.in/cheggaaa/pb.v1"
)

var appVersion = "2.0.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: [flags] <firmware file>\n", os.Args[0])
		flag.PrintDefaults()
	}
	version := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("wally-cli v%s", appVersion))
		os.Exit(0)
	}

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	path := ""
	extension := ""
	if uri, err := url.Parse(flag.Arg(0)); err == nil {
		switch uri.Scheme {
		case "", "file":
			extension = filepath.Ext(uri.Path)
			switch extension {
			case ".bin", ".hex":
				path = uri.Path
			}
		}
	}

	if path == "" {
		fmt.Println("Please provide a valid firmware file: a .hex file (ErgoDox EZ) or a .bin file (Moonlander / Planck EZ)")
		os.Exit(2)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("The file path you specified does not exist:", path)
		os.Exit(1)
	}

	var boardType wallycli.BoardType
	switch extension {
	case ".bin":
		boardType = wallycli.DfuBoard
	case ".hex":
		boardType = wallycli.TeensyBoard
	}

	b, err := wallycli.New(boardType)
	if err != nil {
		log.Fatal(err)
	}

	if err := b.FlashAsync(path); err != nil {
		log.Fatal(err)
	}

	spinner := spin.New("%s Press the reset button of your keyboard.")
	spinner.Start()
	spinnerStopped := false

	var progress *pb.ProgressBar
	progressStarted := false

	for !b.Finished() {
		time.Sleep(500 * time.Millisecond)
		if b.Running() {
			if spinnerStopped == false {
				spinner.Stop()
				spinnerStopped = true
			}
			if progressStarted == false {
				progressStarted = true
				progress = pb.StartNew(b.TotalSteps())
			}
			progress.Set(b.CompletedSteps())
		}
	}
	if progressStarted {
		progress.Finish()
	}

	if b.FlashError() != nil {
		log.Fatal(b.FlashError())
	}

	fmt.Println("Your keyboard was successfully flashed and rebooted. Enjoy the new firmware!")
}
